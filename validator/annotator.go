package validator

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/garethr/kubeval/kubeval"
	"github.com/google/go-github/github"
	"github.com/xeipuuv/gojsonschema"
)

// AnnotateFile takes bytes and a CommitFile and returns CheckRunAnnotations
func AnnotateFile(bytes *[]byte, file *github.CommitFile) []*github.CheckRunAnnotation {
	return AnnotateFileWithSchema(bytes, file, &KubeValidatorConfigSchema{
		Version:    "master",
		BaseURL:    "https://raw.githubusercontent.com/garethr",
		ConfigType: "kubernetes",
		Strict:     false,
	})
}

// AnnotateFileWithSchema takes bytes, a CommitFile, and a
// KubeValidatorConfigSchema and returns CheckRunAnnotations.
func AnnotateFileWithSchema(bytes *[]byte, file *github.CommitFile, config *KubeValidatorConfigSchema) []*github.CheckRunAnnotation {
	var annotations []*github.CheckRunAnnotation
	if config.Version != "" {
		kubeval.Version = config.Version
	}
	if config.BaseURL != "" {
		kubeval.SchemaLocation = config.BaseURL
	}
	kubeval.Strict = config.Strict
	if config.ConfigType == "openstack" {
		kubeval.OpenShift = true
	} else {
		kubeval.OpenShift = false
	}

	var schemaName string
	if config.Name != "" {
		schemaName = config.Name
	} else if config.Version != "" {
		schemaName = config.Version
	} else {
		schemaName = fmt.Sprintf("%v", config)
	}

	results, err := kubeval.Validate(*bytes, file.GetFilename())
	// log.Printf("%+v", results)

	if err != nil {
		annotations = append(annotations, &github.CheckRunAnnotation{
			FileName:     file.Filename,
			BlobHRef:     file.BlobURL,
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String(fmt.Sprintf("Error validating %s against %s schema", results[0].Kind, schemaName)),
			Message:      github.String(fmt.Sprintf("%+v", err)),
		})
		return annotations
	}

	for _, result := range results {
		for _, error := range result.Errors {
			annotations = append(annotations, &github.CheckRunAnnotation{
				FileName:     file.Filename,
				BlobHRef:     file.BlobURL,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("Error validating %s against %s schema", results[0].Kind, schemaName)),
				Message:      github.String(error.String()),
				RawDetails:   github.String(resultErrorDetailString(error)),
			})
		}
	}
	return annotations
}

func resultErrorDetailString(e gojsonschema.ResultError) string {
	details := e.Details()
	var buffer bytes.Buffer
	keys := make([]string, 0, len(details))
	for k := range details {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		buffer.WriteString(fmt.Sprintf("* %s: %s\n", k, details[k]))
	}

	return buffer.String()
}
