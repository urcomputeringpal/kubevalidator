package validator

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
)

func TestAnnotationsForValidCandidates(t *testing.T) {
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
	candidates = append(candidates, *candidate)

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
	candidates = append(candidates, *candidate2)

	annotations := candidates.Validate()

	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(6),
		EndLine:      github.Int(7),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating Deployment against master schema"),
		Message:      github.String("template: template is required"),
		RawDetails:   github.String("* context: (root).spec\n* field: template\n* property: template\n"),
	}, {
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(7),
		EndLine:      github.Int(8),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating Deployment against master schema"),
		Message:      github.String("spec.replicas: Invalid type. Expected: integer, given: string"),
		RawDetails:   github.String("* context: (root).spec.replicas\n* expected: integer\n* field: spec.replicas\n* given: string\n"),
	}}

	if len(annotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(annotations), len(want))
	}

	for i, annotation := range annotations {
		if diff := deep.Equal(annotation, want[i]); diff != nil {
			t.Error(diff)
		}
	}
}
