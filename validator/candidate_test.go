package validator

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
)

func TestAnnotationsForValidCandidate(t *testing.T) {
	candidate := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			Filename: github.String("fixtures/deployment.yaml"),
		}, nil)

	filePath, _ := filepath.Abs("../fixtures/deployment.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	candidate.setBytes(&fileContents)
	annotations := candidate.Validate()

	if len(annotations) > 0 {
		t.Errorf("Expected no annotations, got %+v", github.Stringify(annotations))
	}
}

func TestAnnotationsForInvalidCandidate(t *testing.T) {
	candidate := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			Filename: github.String("deployment.yaml"),
		}, nil)

	filePath, _ := filepath.Abs("../fixtures/invalid.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	candidate.setBytes(&fileContents)
	annotations := candidate.Validate()

	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(6),
		EndLine:      github.Int(8),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating Deployment against master schema"),
		Message:      github.String("template: template is required"),
		RawDetails:   github.String("* context: (root).spec\n* field: template\n* property: template\n"),
	}, {
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(7),
		EndLine:      github.Int(7),
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

func TestAnnotationsWithCustomSchemaSuccess(t *testing.T) {
	schema := &KubeValidatorConfigSchema{
		Version: "1.10.0",
	}
	var schemas []*KubeValidatorConfigSchema
	schemas = append(schemas, schema)
	candidate := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
			Filename: github.String("volumeerror.yaml"),
		}, schemas)

	filePath, _ := filepath.Abs("../fixtures/invalid/1.6.0/volumeerror.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	candidate.setBytes(&fileContents)
	annotations := candidate.Validate()

	if len(annotations) != 0 {
		t.Errorf("%d annotations returned, expected 0: %+v", len(annotations), github.Stringify(annotations))
	}
}

func TestAnnotationsWithCustomSchemaFailure(t *testing.T) {
	schema := &KubeValidatorConfigSchema{
		Version: "1.6.0",
	}
	var schemas []*KubeValidatorConfigSchema
	schemas = append(schemas, schema)
	candidate := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
			Filename: github.String("volumeerror.yaml"),
		}, schemas)

	filePath, _ := filepath.Abs("../fixtures/invalid/1.6.0/volumeerror.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	candidate.setBytes(&fileContents)
	annotations := candidate.Validate()

	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("volumeerror.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating VolumeError against 1.6.0 schema"),
		Message:      github.String("1 error occurred:\n\t* Problem loading schema from the network at https://raw.githubusercontent.com/garethr/kubernetes-json-schema/master/v1.6.0-standalone/volumeerror.json: Could not read schema from HTTP, response status is 404 Not Found\n\n"),
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
