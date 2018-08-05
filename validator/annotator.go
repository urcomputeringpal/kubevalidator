package validator

import (
	"fmt"

	"github.com/garethr/kubeval/kubeval"
	"github.com/google/go-github/github"
)

// AnnotateFile takes bytes and a CommitFile and returns CheckRunAnnotations
func (c *Context) AnnotateFile(bytes *[]byte, file *github.CommitFile) ([]*github.CheckRunAnnotation, error) {
	var annotations []*github.CheckRunAnnotation
	results, err := kubeval.Validate(*bytes, file.GetFilename())
	if err != nil {
		return annotations, err
	}
	for _, result := range results {
		for _, error := range result.Errors {
			annotations = append(annotations, &github.CheckRunAnnotation{
				FileName:     file.Filename,
				BlobHRef:     file.BlobURL,
				StartLine:    github.Int(1),
				EndLine:      github.Int(1),
				WarningLevel: github.String("failure"),
				Title:        github.String(fmt.Sprintf("%s", error.Description())),
				Message:      github.String(fmt.Sprintf("%s", error)),
				RawDetails:   github.String(fmt.Sprintf("%#v", error)),
			})
		}
	}
	return annotations, nil
}
