package validator

import (
	"log"

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
		log.PrintLn(e.Repo.GetOwner())
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

		cfg, skaffoldErr := skaffold.GetConfig([]byte(content), true)
		if skaffoldErr != nil {
			log.Println(errors.Wrap(err, "Couldn't parse skaffold.yaml"))
			return
		}

		skaffoldConfig := cfg.(*skaffold.SkaffoldConfig)

		log.Println(skaffoldConfig)

		// Determine which schema to use

		// Kick off a check run

		// Validate the files

		// Annotate the PR
	}
	return
}
