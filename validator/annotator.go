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
		SchemaFork: "garethr",
		ConfigType: "kubernetes",
		Strict:     false,
	})
}

// AnnotateFileWithSchema takes bytes, a CommitFile, and a
// KubeValidatorConfigSchema and returns CheckRunAnnotations.
func AnnotateFileWithSchema(bytes *[]byte, file *github.CommitFile, schema *KubeValidatorConfigSchema) []*github.CheckRunAnnotation {
	var annotations []*github.CheckRunAnnotation
	kubeval.SchemaLocation = schema.SchemaLocation()

	// TODO move more of this into KubeValidatorConfigSchema
	if schema.Version != "" {
		kubeval.Version = schema.Version
	}

	kubeval.Strict = schema.Strict
	if schema.ConfigType == "openstack" {
		kubeval.OpenShift = true
	} else {
		kubeval.OpenShift = false
	}

	var schemaName string
	if schema.Name != "" {
		schemaName = schema.Name
	} else if schema.Version != "" {
		schemaName = schema.Version
	} else {
		schemaName = fmt.Sprintf("%v", schema)
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
