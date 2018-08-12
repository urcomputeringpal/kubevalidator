package validator

import (
	"fmt"

	"github.com/garethr/kubeval/kubeval"
	"github.com/google/go-github/github"
)

// AnnotateFile takes bytes and a CommitFile and returns CheckRunAnnotations
func AnnotateFile(bytes *[]byte, file *github.CommitFile) ([]*github.CheckRunAnnotation, error) {
	return AnnotateFileWithSchema(bytes, file, &KubeValidatorConfigSchema{
		Version:    "master",
		BaseURL:    "https://raw.githubusercontent.com/garethr",
		ConfigType: "kubernetes",
		Strict:     false,
	})
}

// AnnotateFileWithSchema takes bytes, a CommitFile, and a
// KubeValidatorConfigSchema and returns CheckRunAnnotations.
func AnnotateFileWithSchema(bytes *[]byte, file *github.CommitFile, config *KubeValidatorConfigSchema) ([]*github.CheckRunAnnotation, error) {
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

	results, err := kubeval.Validate(*bytes, file.GetFilename())
	// log.Printf("%+v", results)

	if err != nil {
		annotations = append(annotations, &github.CheckRunAnnotation{
			FileName:     file.Filename,
			BlobHRef:     file.BlobURL,
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String(fmt.Sprintf("Error validating %s", results[0].Kind)),
			Message:      github.String(fmt.Sprintf("%+v", err)),
			RawDetails:   github.String(fmt.Sprintf("%+v", results)),
		})
		return annotations, nil
	}

	for _, result := range results {
		for _, error := range result.Errors {
			annotations = append(annotations, &github.CheckRunAnnotation{
				FileName:     file.Filename,
				BlobHRef:     file.BlobURL,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("%s", error)),
				Message:      github.String(fmt.Sprintf("%+v", error.Details())),
				RawDetails:   github.String(fmt.Sprintf("%s=%s", error.Context().String(), error.Value())),
			})
		}
	}
	return annotations, nil
}
