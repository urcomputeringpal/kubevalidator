package validator

import (
	"log"

	"github.com/pkg/errors"

	skaffold "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/google/go-github/github"
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
		fileContent, _, _, err := c.github.Repositories.GetContents(*c.ctx, *e.Sender.Name, *e.Repo.Name, "skaffold.yaml", &github.RepositoryContentGetOptions{})
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't find skaffold.yaml"))
			return
		}
		content, err := fileContent.GetContent()
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't parse contents"))
			return
		}

		cfg, err := skaffold.GetConfig([]byte(content), true)
		if err != nil {
			log.Println(errors.Wrap(err, "Couldn't parse skaffold.yaml"))
			return
		}

		log.Println(cfg.GetVersion())

		// Determine which schema to use

		// Kick off a check run

		// Validate the files

		// Annotate the PR
	}
	return
}
