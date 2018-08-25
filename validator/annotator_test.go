package validator

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
)

func TestAnnotationsForValidFile(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/deployment.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations := AnnotateFile(&fileContents, &github.CommitFile{
		Filename: github.String("fixtures/deployment.yaml"),
	})

	if len(checkRunAnnotations) > 0 {
		t.Errorf("Expected no annotations, got %+v", *checkRunAnnotations[0].Message)
	}
}

func TestAnnotationsForInvalidFile(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/invalid.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations := AnnotateFile(&fileContents, &github.CommitFile{
		BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		Filename: github.String("deployment.yaml"),
	})
	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating Deployment against master schema"),
		Message:      github.String("template: template is required"),
		RawDetails:   github.String("asdf"),
	}, {
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating Deployment against master schema"),
		Message:      github.String("spec.replicas: Invalid type. Expected: integer, given: string"),
		RawDetails:   github.String("asdf"),
	}}

	if len(checkRunAnnotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(checkRunAnnotations), len(want))
	}

	for i, annotation := range checkRunAnnotations {
		if diff := deep.Equal(annotation, want[i]); diff != nil {
			t.Error(diff)
		}
	}
}

func TestAnnotationsWithCustomSchemaSuccess(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/invalid/1.6.0/volumeerror.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations := AnnotateFileWithSchema(&fileContents,
		&github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
			Filename: github.String("volumeerror.yaml"),
		},
		&KubeValidatorConfigSchema{
			Version: "1.10.0",
		})

	if len(checkRunAnnotations) != 0 {
		t.Errorf("%d annotations returned, expected 0: %+v", len(checkRunAnnotations), checkRunAnnotations[0].GetTitle())
	}
}

func TestAnnotationsWithCustomSchemaFailure(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/invalid/1.6.0/volumeerror.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations := AnnotateFileWithSchema(&fileContents,
		&github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
			Filename: github.String("volumeerror.yaml"),
		},
		&KubeValidatorConfigSchema{
			Version: "1.6.0",
		})
	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("volumeerror.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating VolumeError against 1.6.0 schema"),
		Message:      github.String("1 error occurred:\n\t* Problem loading schema from the network at https://raw.githubusercontent.com/garethr/kubernetes-json-schema/master/v1.6.0-standalone/volumeerror.json: Could not read schema from HTTP, response status is 404 Not Found\n\n"),
	}}

	if len(checkRunAnnotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(checkRunAnnotations), len(want))
	}

	for i, annotation := range checkRunAnnotations {
		if diff := deep.Equal(annotation, want[i]); diff != nil {
			t.Error(diff)
		}
	}
}
