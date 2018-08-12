package validator

import (
	"fmt"
	"log"
	"path"
	"time"

	skaffold "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Process handles webhook events kinda like Probot does
func (c *Context) Process() {
	switch e := c.event.(type) {
	case *github.CheckSuiteEvent:
		c.ProcessCheckSuite(c.event.(*github.CheckSuiteEvent))
		return
	default:
		log.Printf("ignoring %s\n", e)
		return
	}
}

// ProcessCheckSuite validates the Kubernetes YAML that has changed on checks
// associated with PRs.
func (c *Context) ProcessCheckSuite(e *github.CheckSuiteEvent) {
	if *e.Action == "requested" || *e.Action == "re-requested" {

		// Kick off a check run
		checkRunStart := time.Now()
		checkRunStatus := "in_progress"
		checkRunTitle := "kubevalidator"
		checkRunSummary := "Validating Kubernetes YAML"
		checkRunOpt := github.CreateCheckRunOptions{
			Name:       checkRunTitle,
			HeadBranch: e.CheckSuite.GetHeadBranch(),
			HeadSHA:    e.CheckSuite.GetHeadSHA(),
			Status:     &checkRunStatus,
			StartedAt:  &github.Timestamp{checkRunStart},
			Output: &github.CheckRunOutput{
				Title:   &checkRunTitle,
				Summary: &checkRunSummary,
			},
		}

		_, _, checkRunErr := c.github.Checks.CreateCheckRun(*c.ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
		if checkRunErr != nil {
			log.Println(errors.Wrap(checkRunErr, "Couldn't create check run"))
			return
		}

		// Determine which files to load
		fileContent, _, _, err := c.github.Repositories.GetContents(*c.ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), "skaffold.yaml", &github.RepositoryContentGetOptions{
			Ref: e.CheckSuite.GetHeadSHA(),
		})
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't find skaffold.yaml"))
			return
		}

		content, err := fileContent.GetContent()
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't load contents"))
			return
		}

		apiVersion := &skaffold.APIVersion{}
		err = yaml.Unmarshal([]byte(content), apiVersion)
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't parse api version out of skaffold.yaml"))
			return
		}

		if apiVersion.Version != skaffold.LatestVersion {
			log.Println(errors.New("skaffold.yaml out of date: run `skaffold fix`"))
			return
		}

		cfg, err := skaffold.GetConfig([]byte(content), true)
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't parse skaffold.yaml"))
			return
		}

		skaffoldConfig := cfg.(*skaffold.SkaffoldConfig)

		if skaffoldConfig.Deploy.DeployType.KubectlDeploy == nil {
			log.Println(errors.New("Couldn't find kubectl manifests in skaffold.yaml"))
			return
		}

		filesToValidate := make(map[string]*github.CommitFile)
		for _, pr := range e.CheckSuite.PullRequests {
			files, _, err := c.github.PullRequests.ListFiles(*c.ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), pr.GetNumber(), &github.ListOptions{})
			if err != nil {
				log.Println(errors.Wrap(err, "Couldn't list files"))
				return
			}
			for _, file := range files {
				for _, pattern := range skaffoldConfig.Deploy.DeployType.KubectlDeploy.Manifests {
					if matched, _ := path.Match(pattern, file.GetFilename()); matched {
						filesToValidate[file.GetFilename()] = file
					}
				}
			}
		}

		// Determine which schema to use
		var schema *KubeValidatorConfigSchema
		var configSpec *KubeValidatorConfigSpec
		schemaFile, _, _, err := c.github.Repositories.GetContents(*c.ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), ".github/kubevalidator.yaml", &github.RepositoryContentGetOptions{
			Ref: e.CheckSuite.GetHeadSHA(),
		})
		if err == nil {
			schema = &KubeValidatorConfigSchema{}
		} else {
			schemaContent, err := schemaFile.GetContent()
			if err != nil {
				log.Println(errors.Wrap(err, "Couldn't load contents"))
				return
			}

			// unmarshal schemaFile into a KubeValidatorConfig and eventually a
			// KubeValidatorConfigSchema
			var config *KubeValidatorConfig
			err = yaml.Unmarshal([]byte(schemaContent), config)
			if err != nil {
				log.Fatalf("Couldn't unmarshal .github/kubevalidator.yaml: %v", err)
			}
		}

		// Validate the files
		var annotations []*github.CheckRunAnnotation
		for filename, file := range filesToValidate {
			fileToValidate, _, _, err := c.github.Repositories.GetContents(*c.ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), filename, &github.RepositoryContentGetOptions{
				Ref: e.CheckSuite.GetHeadSHA(),
			})
			if err != nil {
				log.Println(errors.Wrap(err, "Couldn't load file"))
				return
			}

			contentToValidate, err := fileToValidate.GetContent()
			if err != nil {
				log.Println(errors.Wrap(err, "Couldn't load contents"))
				return
			}

			if schema == nil && configSpec != nil {
				for _, manifestConfig := range configSpec.manifests {
					if matched, _ := path.Match(manifestConfig.glob, file.GetFilename()); matched {
						schema = manifestConfig.schemas[0]
					}
				}
			}

			bytes := []byte(contentToValidate)
			fileAnnotations, err := AnnotateFileWithSchema(&bytes, file, schema)
			if err != nil {
				log.Println(errors.Wrap(err, "Couldn't validate file"))
				return
			}
			annotations = append(annotations, fileAnnotations...)
		}

		// Annotate the PR
		checkRunStatus = "completed"
		var checkRunConclusion string
		var checkRunText string
		if len(filesToValidate) == 0 {
			checkRunConclusion = "neutral"
			checkRunText = "no files matched"
		} else {
			if len(annotations) > 0 {
				checkRunConclusion = "failure"
			} else {
				checkRunConclusion = "success"
			}
			checkRunText = fmt.Sprintf("%d files checked, %d errors", len(filesToValidate), len(annotations))
		}

		checkRunOpt = github.CreateCheckRunOptions{
			Name:        checkRunTitle,
			HeadBranch:  e.CheckSuite.GetHeadBranch(),
			HeadSHA:     e.CheckSuite.GetHeadSHA(),
			Status:      &checkRunStatus,
			Conclusion:  &checkRunConclusion,
			StartedAt:   &github.Timestamp{checkRunStart},
			CompletedAt: &github.Timestamp{time.Now()},
			Output: &github.CheckRunOutput{
				Title:       &checkRunTitle,
				Summary:     &checkRunSummary,
				Text:        &checkRunText,
				Annotations: annotations,
			},
		}

		_, _, finalCheckRunErr := c.github.Checks.CreateCheckRun(*c.ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
		if finalCheckRunErr != nil {
			log.Println(errors.Wrap(finalCheckRunErr, "Couldn't create check run"))
			return
		}
	}
	return
}
