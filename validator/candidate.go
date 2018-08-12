package validator

import (
	"fmt"

	"github.com/google/go-github/github"
)

// Candidate reprensets a file to be validated
type Candidate struct {
	File    *github.CommitFile
	Schemas []*KubeValidatorConfigSchema
}

// MarkdownListItem returns a string that represents the Candidate designed for
// use in a Markdown List
func (c *Candidate) MarkdownListItem() string {
	return fmt.Sprintf("* [%s](%s)", c.File.GetFilename(), c.File.GetBlobURL())
}
