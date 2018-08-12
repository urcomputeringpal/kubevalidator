package validator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

// Context contains an event payload an a configured client
type Context struct {
	event  interface{}
	github *github.Client
	ctx    *context.Context
}

type schemaMap struct {
	file    *github.CommitFile
	schemas []*KubeValidatorConfigSchema
}

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
		createCheckRunErr := c.createInitialCheckRun(e)
		if createCheckRunErr != nil {
			log.Println(errors.Wrap(createCheckRunErr, "Couldn't create check run"))
			return
		}

		checkRunStart := time.Now()

		// Determine which files to validate
		filesToValidate, fileSchemaMapError := c.buildFileSchemaMap(e)
		if fileSchemaMapError != nil {
			log.Println(fileSchemaMapError)
			return
		}

		// Validate the files
		var annotations []*github.CheckRunAnnotation
		for filename, file := range filesToValidate {
			bytes, err := c.bytesForFilename(e, filename)
			if err != nil {
				// TODO add an annotation on the file instead
				log.Println(err)
				return
			}

			fileAnnotations, err := AnnotateFileWithSchema(bytes, file.file, file.schemas[0])
			if err != nil {
				// TODO add an annotation on the file instead
				log.Println(errors.Wrap(err, fmt.Sprintf("Error validating %s", filename)))
				return
			}
			annotations = append(annotations, fileAnnotations...)
		}

		// Annotate the PR
		finalCheckRunErr := c.createFinalCheckRun(&checkRunStart, e, len(filesToValidate), annotations)
		if finalCheckRunErr != nil {
			log.Println(errors.Wrap(finalCheckRunErr, "Couldn't create check run"))
			return
		}
	}
	return
}
