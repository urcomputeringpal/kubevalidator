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
	Event  interface{}
	Github *github.Client
	Ctx    *context.Context
}

type schemaMap struct {
	File    *github.CommitFile
	Schemas []*KubeValidatorConfigSchema
}

// Process handles webhook events kinda like Probot does
func (c *Context) Process() {
	switch e := c.Event.(type) {
	case *github.CheckSuiteEvent:
		c.ProcessCheckSuite(c.Event.(*github.CheckSuiteEvent))
		return
	// case *github.PullRequestEvent:
	// TODO Request a check suite when a PR is opened
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
			// TODO return a 500 to signal that retry is preferred
			log.Println(errors.Wrap(createCheckRunErr, "Couldn't create check run"))
			return
		}

		checkRunStart := time.Now()
		var annotations []*github.CheckRunAnnotation

		// Determine which files to validate
		filesToValidate, configAnnotation, fileSchemaMapError := c.buildFileSchemaMap(e)
		if fileSchemaMapError != nil {
			// TODO fail the checkrun instead
			log.Println(fileSchemaMapError)
			return
		}
		if configAnnotation != nil {
			annotations = append(annotations, configAnnotation)
		}

		// Validate the files
		for filename, file := range filesToValidate {
			bytes, err := c.bytesForFilename(e, filename)
			if err != nil {
				annotations = append(annotations, &github.CheckRunAnnotation{
					FileName:     file.File.Filename,
					BlobHRef:     file.File.BlobURL,
					StartLine:    github.Int(1),
					EndLine:      github.Int(1),
					WarningLevel: github.String("failure"),
					Title:        github.String(fmt.Sprintf("Error loading %s from GitHub", file.File.Filename)),
					Message:      github.String(fmt.Sprintf("%+v", err)),
				})
			}

			if file.Schemas == nil {
				fileAnnotations, err := AnnotateFile(bytes, file.File)
				if err != nil {
					annotations = append(annotations, &github.CheckRunAnnotation{
						FileName:     file.File.Filename,
						BlobHRef:     file.File.BlobURL,
						StartLine:    github.Int(1),
						EndLine:      github.Int(1),
						WarningLevel: github.String("failure"),
						Title:        github.String(fmt.Sprintf("Error validating %s", file.File.Filename)),
						Message:      github.String(fmt.Sprintf("%+v", err)),
					})
				}
				annotations = append(annotations, fileAnnotations...)
			}

			for _, schema := range file.Schemas {
				fileAnnotations, err := AnnotateFileWithSchema(bytes, file.File, schema)
				if err != nil {
					var schemaName string
					if schema.Name != "" {
						schemaName = schema.Name
					} else {
						schemaName = fmt.Sprintf("%v", schema)
					}
					annotations = append(annotations, &github.CheckRunAnnotation{
						FileName:     file.File.Filename,
						BlobHRef:     file.File.BlobURL,
						StartLine:    github.Int(1),
						EndLine:      github.Int(1),
						WarningLevel: github.String("failure"),
						Title:        github.String(fmt.Sprintf("Error validating %s using %s schema", file.File.Filename, schemaName)),
						Message:      github.String(fmt.Sprintf("%+v", err)),
					})
				}
				annotations = append(annotations, fileAnnotations...)
			}

		}

		// Annotate the PR
		finalCheckRunErr := c.createFinalCheckRun(&checkRunStart, e, len(filesToValidate), annotations)
		if finalCheckRunErr != nil {
			// TODO return a 500 to signal that retry is preferred
			log.Println(errors.Wrap(finalCheckRunErr, "Couldn't create check run"))
			return
		}
	}
	return
}
