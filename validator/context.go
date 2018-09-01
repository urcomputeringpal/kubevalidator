package validator

import (
	"context"
	"log"
	"reflect"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

// Context contains an event payload an a configured client
type Context struct {
	Event  interface{}
	Github *github.Client
	Ctx    *context.Context
	AppID  *int
}

// Process handles webhook events kinda like Probot does
func (c *Context) Process() {
	switch e := c.Event.(type) {
	case *github.CheckSuiteEvent:
		c.ProcessCheckSuite(c.Event.(*github.CheckSuiteEvent))
		return
	case *github.PullRequestEvent:
		prEvent := c.Event.(*github.PullRequestEvent)
		if *prEvent.Action == "opened" {
			results, _, err := c.Github.Checks.ListCheckSuitesForRef(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), prEvent.PullRequest.Head.GetRef(), &github.ListCheckSuiteOptions{
				AppID: c.AppID,
			})
			if err != nil {
				log.Printf("%+v\n", err)
			}
			if results.GetTotal() == 1 {
				suite := results.CheckSuites[0]
				_, err := c.Github.Checks.ReRequestCheckSuite(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), suite.GetID())
				if err != nil {
					log.Printf("%+v\n", err)
				}
			}
		}
	default:
		log.Printf("ignoring %s\n", reflect.TypeOf(e).String())
		return
	}
}

// ProcessCheckSuite validates the Kubernetes YAML that has changed on checks
// associated with PRs.
func (c *Context) ProcessCheckSuite(e *github.CheckSuiteEvent) {
	if *e.Action == "created" || *e.Action == "requested" || *e.Action == "rerequested" {
		createCheckRunErr := c.createInitialCheckRun(e)
		if createCheckRunErr != nil {
			// TODO return a 500 to signal that retry is preferred
			log.Println(errors.Wrap(createCheckRunErr, "Couldn't create check run"))
			return
		}

		checkRunStart := time.Now()
		var annotations []*github.CheckRunAnnotation
		var candidates Candidates

		config, configAnnotation, err := c.kubeValidatorConfigOrAnnotation(e)
		if err != nil {
			c.createConfigMissingCheckRun(&checkRunStart, e)
			return
		}
		if configAnnotation != nil {
			annotations = append(annotations, configAnnotation)
			c.createConfigInvalidCheckRun(&checkRunStart, e, annotations)
			return
		}

		// Determine which files to validate
		changedFileList, fileListError := c.changedFileList(e)
		if fileListError != nil {
			// TODO fail the checkrun instead
			log.Println(fileListError)
			return
		}

		candidates = config.matchingCandidates(c, changedFileList)
		annotations = append(annotations, candidates.LoadBytes()...)
		annotations = append(annotations, candidates.Validate()...)

		// Annotate the PR
		finalCheckRunErr := c.createFinalCheckRun(&checkRunStart, e, candidates, annotations)
		if finalCheckRunErr != nil {
			// TODO return a 500 to signal that retry is preferred
			log.Println(errors.Wrap(finalCheckRunErr, "Couldn't create check run"))
			return
		}
	}
	return
}
