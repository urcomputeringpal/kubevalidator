package validator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
)

func TestAnnotationsForInvalidCandidates(t *testing.T) {
	var candidates Candidates
	candidate := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			Filename: github.String("/deployment.yaml"),
		}, nil)

	filePath, _ := filepath.Abs("../fixtures/deployment.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	candidate.setBytes(&fileContents)
	candidates = append(candidates, candidate)

	candidate2 := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			Filename: github.String("deployment.yaml"),
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		}, nil)

	filePath2, _ := filepath.Abs("../fixtures/invalid.yaml")
	fileContents2, _ := ioutil.ReadFile(filePath2)
	candidate2.setBytes(&fileContents2)
	candidates = append(candidates, candidate2)

	annotations := candidates.Validate()

	want := []*github.CheckRunAnnotation{
		{
			FileName:     github.String("deployment.yaml"),
			BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String("Error validating Deployment against master schema"),
			Message:      github.String("spec.replicas: Invalid type. Expected: integer, given: string"),
			RawDetails:   github.String("* context: (root).spec.replicas\n* expected: integer\n* field: spec.replicas\n* given: string\n"),
		},
		{
			FileName:     github.String("deployment.yaml"),
			BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String("Error validating Deployment against master schema"),
			Message:      github.String("template: template is required"),
			RawDetails:   github.String("* context: (root).spec\n* field: template\n* property: template\n"),
		},
	}

	if len(annotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(annotations), len(want))
	}

	for i, annotation := range annotations {
		if diff := deep.Equal(annotation, want[i]); diff != nil {
			t.Error(diff)
		}
	}
}

func TestLoadingCandidatesBytesFromGitHub(t *testing.T) {
	client, mux, _, teardown := setup()
	filePath, _ := filepath.Abs("../fixtures/invalid/deployment/multiple.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	contentString := base64.StdEncoding.EncodeToString(fileContents)
	defer teardown()
	mux.HandleFunc("/repos/r/o/contents/deployment.yaml", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, `{
			"type": "file",
			"encoding": "base64",
			"size": 20678,
			"name": "LICENSE",
			"path": "LICENSE",
			"content": "%s"
		}`, contentString)
	})

	ctx := context.Background()

	schema := &KubeValidatorConfigSchema{
		Strict: true,
	}
	var schemas []*KubeValidatorConfigSchema
	schemas = append(schemas, schema)

	candidate := NewCandidate(
		&Context{
			Ctx: &ctx,
			Event: &github.CheckSuiteEvent{
				CheckSuite: &github.CheckSuite{
					HeadSHA: github.String("master"),
				},
				Repo: &github.Repository{
					Name: github.String("o"),
					Owner: &github.User{
						Login: github.String("r"),
					},
				},
			},
			Github: client,
		}, &github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			Filename: github.String("deployment.yaml"),
		}, schemas)

	var annotations Annotations

	var candidates Candidates
	candidates = append(candidates, candidate)

	annotations = append(annotations, candidates.LoadBytes()...)
	annotations = append(annotations, candidates.Validate()...)

	var want Annotations
	want = []*github.CheckRunAnnotation{
		{
			FileName:     github.String("deployment.yaml"),
			BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String("Error validating Deployment against strict schema"),
			Message:      github.String("extra: Additional property extra is not allowed"),
			RawDetails:   github.String("* context: (root).spec\n* field: extra\n* property: extra\n"),
		},
		{
			FileName:     github.String("deployment.yaml"),
			BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String("Error validating Deployment against strict schema"),
			Message:      github.String("spec.replicas: Invalid type. Expected: integer, given: string"),
			RawDetails:   github.String("* context: (root).spec.replicas\n* expected: integer\n* field: spec.replicas\n* given: string\n"),
		},
		{
			FileName:     github.String("deployment.yaml"),
			BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			StartLine:    github.Int(1),
			EndLine:      github.Int(1),
			WarningLevel: github.String("failure"),
			Title:        github.String("Error validating Deployment against strict schema"),
			Message:      github.String("extra-container: Additional property extra-container is not allowed"),
			RawDetails:   github.String("* context: (root).spec.template.spec.containers.0\n* field: extra-container\n* property: extra-container\n"),
		},
	}

	sort.Sort(want)
	if len(annotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(annotations), len(want))
	}

	if diff := deep.Equal(annotations, want); diff != nil {
		t.Error(diff)
	}
	return
}

func TestLineNumbers(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	filepath.Walk("../fixtures/line-numbers", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {

			filePath, _ := filepath.Abs(path)
			fileBase := filepath.Base(path)
			fileContents, _ := ioutil.ReadFile(filePath)
			contentString := base64.StdEncoding.EncodeToString(fileContents)
			mux.HandleFunc(fmt.Sprintf("/repos/r/o/contents/%s", fileBase), func(w http.ResponseWriter, r *http.Request) {
				testMethod(t, r, "GET")
				fmt.Fprintf(w, `{
					"type": "file",
					"encoding": "base64",
					"size": 20678,
					"name": "%s",
					"path": "%s",
					"content": "%s"
				}`, fileBase, fileBase, contentString)
			})

			ctx := context.Background()

			schema := &KubeValidatorConfigSchema{
				Strict:      true,
				LineNumbers: true,
			}
			var schemas []*KubeValidatorConfigSchema
			schemas = append(schemas, schema)

			candidate := NewCandidate(
				&Context{
					Ctx: &ctx,
					Event: &github.CheckSuiteEvent{
						CheckSuite: &github.CheckSuite{
							HeadSHA: github.String("master"),
						},
						Repo: &github.Repository{
							Name: github.String("o"),
							Owner: &github.User{
								Login: github.String("r"),
							},
						},
					},
					Github: client,
				}, &github.CommitFile{
					BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
					Filename: github.String(fileBase),
				}, schemas)

			var annotations Annotations

			var candidates Candidates
			candidates = append(candidates, candidate)

			annotations = append(annotations, candidates.LoadBytes()...)
			annotations = append(annotations, candidates.Validate()...)

			scanner := bufio.NewScanner(bytes.NewReader(fileContents))
			var comment string
			for scanner.Scan() {
				comment = scanner.Text()
				break
			}

			matches := strings.Split(comment, " ")

			if len(annotations) != (len(matches) - 1) {
				t.Errorf("%s: expected %d annotations, got %d", path, (len(matches) - 1), len(annotations))
				return nil
			}

			for i, val := range matches[1:] {
				ln := strings.Split(val, "-")
				startLine, _ := strconv.Atoi(ln[0])
				endLine, _ := strconv.Atoi(ln[1])
				if annotations[i].GetStartLine() != startLine || annotations[i].GetEndLine() != endLine {
					t.Errorf("%s[%d]: (%s) expected annotation on lines %d-%d, got %d-%d", path, i, annotations[i].GetMessage(), startLine, endLine, annotations[i].GetStartLine(), annotations[i].GetEndLine())
				}
			}

		}
		return nil
	})
}
