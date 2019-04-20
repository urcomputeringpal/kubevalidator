package validator

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/instrumenta/kubeval/kubeval"
	yamlpatch "github.com/krishicks/yaml-patch"
	difflib "github.com/pmezard/go-difflib/difflib"
	"github.com/xeipuuv/gojsonschema"
	"sourcegraph.com/sourcegraph/go-diff/diff"
)

// Candidate reprensets a file to be validated
type Candidate struct {
	bytes   *[]byte
	context *Context
	file    *github.CommitFile
	schemas []*KubeValidatorConfigSchema
}

const (
	placeholderString = "AAA___KUBEVALIDATOR___PLACEHOLDER___AAA"
)

var (
	defaultSchema = &KubeValidatorConfigSchema{
		Version:    "master",
		SchemaFork: "garethr",
		ConfigType: "kubernetes",
	}
)

// NewCandidate initializes a validation Candidate
func NewCandidate(context *Context, file *github.CommitFile, schemas []*KubeValidatorConfigSchema) *Candidate {
	if len(schemas) == 0 {
		schemas = append(schemas, defaultSchema)
	}
	return &Candidate{
		context: context,
		file:    file,
		schemas: schemas,
	}
}

func (c *Candidate) setBytes(b *[]byte) {
	c.bytes = b
}

// LoadBytes hydrates bytes from GitHub and returns a CheckRunAnnotation if
// an error is encountered
func (c *Candidate) LoadBytes() *github.CheckRunAnnotation {
	b, err := c.context.bytesForFilename(c.context.Event.(*github.CheckSuiteEvent), c.file.GetFilename())
	if err != nil {
		return &github.CheckRunAnnotation{
			Path:            c.file.Filename,
			BlobHRef:        c.file.BlobURL,
			StartLine:       github.Int(1),
			EndLine:         github.Int(1),
			AnnotationLevel: github.String("failure"),
			Title:           github.String(fmt.Sprintf("Error loading %s", c.file.GetFilename())),
			Message:         github.String(fmt.Sprintf("%+v", err)),
		}
	}

	c.bytes = b
	return nil
}

// MarkdownListItem returns a string that represents the Candidate designed for
// use in a Markdown List
func (c *Candidate) MarkdownListItem() string {
	return fmt.Sprintf("* [`./%s`](%s)", c.file.GetFilename(), c.file.GetBlobURL())
}

// Validate bytes with kubeval and return an array of CheckRunAnnotation
func (c *Candidate) Validate() Annotations {
	var annotations Annotations
	for _, schema := range c.schemas {
		kubeval.SchemaLocation = schema.SchemaLocation()

		// TODO move more of this into KubeValidatorConfigSchema
		if schema.Version != "" {
			kubeval.Version = schema.Version
		}

		// TODO configurable
		kubeval.Strict = true
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
			schemaName = "default"
		}

		if c.bytes == nil {
			annotations = append(annotations, &github.CheckRunAnnotation{
				Path:            c.file.Filename,
				BlobHRef:        c.file.BlobURL,
				StartLine:       github.Int(1),
				EndLine:         github.Int(1),
				AnnotationLevel: github.String("failure"),
				Title:           github.String("Candidate has no bytes?"),
				Message:         github.String(fmt.Sprintf("%+v", c)),
			})
			continue
		}

		results, err := kubeval.Validate(*c.bytes, c.file.GetFilename())

		if err != nil {
			annotations = append(annotations, &github.CheckRunAnnotation{
				Path:            c.file.Filename,
				BlobHRef:        c.file.BlobURL,
				StartLine:       github.Int(1),
				EndLine:         github.Int(1),
				AnnotationLevel: github.String("failure"),
				Title:           github.String(fmt.Sprintf("Error validating %s against %s schema", results[0].Kind, schemaName)),
				Message:         github.String(fmt.Sprintf("%+v", err)),
			})
			continue
		}

		for _, result := range results {
			for _, error := range result.Errors {
				startLine := 1
				endLine := 1
				if schema.LineNumbers == true {
					switch error.Type() {
					default:
						// fmt.Println(error.Type())
						startLine, endLine = detectLineNumbersDefault(c.bytes, error)
					}
				}

				annotations = append(annotations, &github.CheckRunAnnotation{
					Path:            c.file.Filename,
					BlobHRef:        c.file.BlobURL,
					StartLine:       &startLine,
					EndLine:         &endLine,
					AnnotationLevel: github.String("failure"),
					Title:           github.String(fmt.Sprintf("Error validating %s against %s schema", result.Kind, schemaName)),
					Message:         github.String(error.String()),
					RawDetails:      github.String(resultErrorDetailString(error)),
				})
			}
		}
	}
	sort.Sort(annotations)
	return annotations
}

func detectLineNumbersDefault(b *[]byte, e gojsonschema.ResultError) (int, int) {
	var dotted string
	rootContext := strings.TrimPrefix(e.Context().String(), "(root).")
	dotted = fmt.Sprintf(".%s", rootContext)
	path := yamlpatch.OpPath(strings.Replace(dotted, ".", "/", -1))
	// log.Println(e.String())
	// log.Println(e.Type())
	// log.Println(path)
	var patch yamlpatch.Patch
	var s interface{}
	s = placeholderString
	value := yamlpatch.NewNode(&s)
	operation := yamlpatch.Operation{
		Op:    "replace",
		Path:  path,
		Value: value,
	}
	patch = append(patch, operation)
	patchedBytes, err := patch.Apply(*b)
	if err != nil {
		return 1, 1
	}

	difflibDiff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(*b)),
		B:        difflib.SplitLines(string(patchedBytes)),
		FromFile: "Original",
		ToFile:   "Current",
		Context:  0,
	}
	unifiedDiffString, err := difflib.GetUnifiedDiffString(difflibDiff)
	if err != nil {
		return 1, 1
	}

	// log.Println(unifiedDiffString)
	fileDiff, err := diff.ParseFileDiff([]byte(unifiedDiffString))
	if err != nil {
		return 1, 1
	}

	for _, hunk := range fileDiff.Hunks {
		scanner := bufio.NewScanner(bytes.NewReader(hunk.Body))

		line := 1
		found := false
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), placeholderString) {
				found = true
				continue
			}
			line++
		}
		if found {
			// log.Printf("%+v", hunk)
			startLine := int(hunk.NewStartLine)
			endLine := int(hunk.NewStartLine + hunk.NewLines)
			// log.Printf("start: %d end: %d", startLine, endLine)

			// if e.Type() == "additional_property_not_allowed" {
			// 	return line, line+1
			// }
			return startLine, endLine
		}

		if err := scanner.Err(); err != nil {
			continue
		}
	}
	return 1, 1
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
