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

const (
	checkRunTitle   = "kubevalidator"
	checkRunSummary = "Validating Kubernetes YAML"
)

// createInitialCheckRun contains the logic which sets the title and summary
// of the check
func (c *Context) createInitialCheckRun(e *github.CheckSuiteEvent) error {
	checkRunStart := time.Now()
	checkRunStatus := "in_progress"

	crt := checkRunTitle
	crs := checkRunSummary
	checkRunOpt := github.CreateCheckRunOptions{
		Name:       checkRunTitle,
		HeadBranch: e.CheckSuite.GetHeadBranch(),
		HeadSHA:    e.CheckSuite.GetHeadSHA(),
		Status:     &checkRunStatus,
		StartedAt:  &github.Timestamp{checkRunStart},
		Output: &github.CheckRunOutput{
			Title:   &crt,
			Summary: &crs,
		},
	}

	_, _, err := c.Github.Checks.CreateCheckRun(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
	if err != nil {
		log.Println(errors.Wrap(err, "Couldn't create check run"))
		return err
	}
	return nil
}

// createFinalCheckRun concludes the check run
func (c *Context) createFinalCheckRun(startedAt *time.Time, e *github.CheckSuiteEvent, numFiles int, annotations []*github.CheckRunAnnotation) error {
	checkRunStatus := "completed"
	var checkRunConclusion string
	var checkRunText string
	if numFiles == 0 {
		checkRunConclusion = "neutral"
		checkRunText = "no files matched"
	} else {
		if len(annotations) > 0 {
			checkRunConclusion = "failure"
		} else {
			checkRunConclusion = "success"
		}
		checkRunText = fmt.Sprintf("%d files checked, %d errors", numFiles, len(annotations))
	}

	crt := checkRunTitle
	crs := checkRunSummary
	checkRunOpt := github.CreateCheckRunOptions{
		Name:        checkRunTitle,
		HeadBranch:  e.CheckSuite.GetHeadBranch(),
		HeadSHA:     e.CheckSuite.GetHeadSHA(),
		Status:      &checkRunStatus,
		Conclusion:  &checkRunConclusion,
		StartedAt:   &github.Timestamp{*startedAt},
		CompletedAt: &github.Timestamp{time.Now()},
		Output: &github.CheckRunOutput{
			Title:       &crt,
			Summary:     &crs,
			Text:        &checkRunText,
			Annotations: annotations,
		},
	}

	_, _, err := c.Github.Checks.CreateCheckRun(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
	if err != nil {
		log.Println(errors.Wrap(err, "Couldn't create check run"))
		return err
	}
	return nil
}

func (c *Context) bytesForFilename(e *github.CheckSuiteEvent, f string) (*[]byte, error) {
	fileToValidate, _, _, err := c.Github.Repositories.GetContents(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), f, &github.RepositoryContentGetOptions{
		Ref: e.CheckSuite.GetHeadSHA(),
	})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Couldn't load %s", f))
	}

	contentToValidate, err := fileToValidate.GetContent()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Couldn't load contents of %s", f))
	}

	bytes := []byte(contentToValidate)
	return &bytes, nil
}

func (c *Context) buildFileSchemaMap(e *github.CheckSuiteEvent) (map[string]*schemaMap, *github.CheckRunAnnotation, error) {
	skaffoldFilename := "skaffold.yaml"
	skaffoldBytes, _ := c.bytesForFilename(e, skaffoldFilename)
	skaffoldBlobHRef := fmt.Sprintf("%s/%s/%s/blob/%s/%s", c.Github.BaseURL, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), e.CheckSuite.GetHeadSHA(), skaffoldFilename)
	var skaffoldConfig *skaffold.SkaffoldConfig
	if skaffoldBytes != nil {
		apiVersion := &skaffold.APIVersion{}
		err := yaml.Unmarshal(*skaffoldBytes, apiVersion)
		if err != nil {
			return nil, &github.CheckRunAnnotation{
				FileName:     &skaffoldFilename,
				BlobHRef:     &skaffoldBlobHRef,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("Couldn't unmarshal %s", skaffoldFilename)),
				Message:      github.String(fmt.Sprintf("%+v", err)),
			}, nil
		}

		if apiVersion.Version != skaffold.LatestVersion {
			// TODO bubble up into check run
			return nil, &github.CheckRunAnnotation{
				FileName:     &skaffoldFilename,
				BlobHRef:     &skaffoldBlobHRef,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("%s out of date", skaffoldFilename)),
				Message:      github.String("Run 'skaffold fix'"),
			}, nil
		}

		cfg, err := skaffold.GetConfig(*skaffoldBytes, true)
		if err != nil {
			return nil, &github.CheckRunAnnotation{
				FileName:     &skaffoldFilename,
				BlobHRef:     &skaffoldBlobHRef,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("Couldn't parse %s", skaffoldFilename)),
				Message:      github.String(fmt.Sprintf("%+v", err)),
			}, nil
		}

		skaffoldConfig = cfg.(*skaffold.SkaffoldConfig)
	}

	var configSpec *KubeValidatorConfigSpec
	configFileName := ".github/kubevalidator.yaml"
	configBlobHRef := fmt.Sprintf("https://%/%s/%s/blob/%s/%s", c.Github.BaseURL, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), e.CheckSuite.GetHeadSHA(), configFileName)
	configBytes, _ := c.bytesForFilename(e, configFileName)
	if configBytes != nil {
		var config *KubeValidatorConfig
		err := yaml.Unmarshal(*configBytes, config)
		if err != nil {
			return nil, &github.CheckRunAnnotation{
				FileName:     &configFileName,
				BlobHRef:     &configBlobHRef,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("Couldn't unmarshal %s", configFileName)),
				Message:      github.String(fmt.Sprintf("%+v", err)),
			}, nil
		}
	}

	filesToValidate := make(map[string]*schemaMap)
	for _, pr := range e.CheckSuite.PullRequests {
		files, _, prListErr := c.Github.PullRequests.ListFiles(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), pr.GetNumber(), &github.ListOptions{})
		if prListErr != nil {
			return nil, nil, errors.Wrap(prListErr, "Couldn't list files")
		}
		for _, file := range files {

			if configSpec != nil {
				for _, manifestConfig := range configSpec.Manifests {
					if matched, _ := path.Match(manifestConfig.Glob, file.GetFilename()); matched {
						filesToValidate[file.GetFilename()] = &schemaMap{
							File:    file,
							Schemas: manifestConfig.Schemas,
						}
					}
				}
			}

			// Append files that match skaffold with a default schema
			for _, pattern := range skaffoldConfig.Deploy.DeployType.KubectlDeploy.Manifests {
				if matched, _ := path.Match(pattern, file.GetFilename()); matched {
					if filesToValidate[file.GetFilename()] == nil {
						filesToValidate[file.GetFilename()] = &schemaMap{File: file}
					}
				}
			}
		}
	}

	return filesToValidate, nil, nil
}
