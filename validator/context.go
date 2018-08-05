package validator

import (
	"fmt"
	"log"
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
		}
		log.Println(skaffoldConfig.Deploy.DeployType.KubectlDeploy.Manifests)

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

		// Determine which schema to use

		// Validate the files

		// Annotate the PR
		checkRunStatus = "completed"
		checkRunConclusion := "neutral"
		checkRunText := fmt.Sprintf("TODO: validate `%s`", skaffoldConfig.Deploy.DeployType.KubectlDeploy.Manifests)
		checkRunOpt = github.CreateCheckRunOptions{
			Name:        checkRunTitle,
			HeadBranch:  e.CheckSuite.GetHeadBranch(),
			HeadSHA:     e.CheckSuite.GetHeadSHA(),
			Status:      &checkRunStatus,
			Conclusion:  &checkRunConclusion,
			StartedAt:   &github.Timestamp{checkRunStart},
			CompletedAt: &github.Timestamp{time.Now()},
			Output: &github.CheckRunOutput{
				Title:   &checkRunTitle,
				Summary: &checkRunSummary,
				Text:    &checkRunText,
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
