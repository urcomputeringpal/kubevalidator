package validator

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
)

func TestAnnotationsForValidFile(t *testing.T) {
	context := &Context{}
	filePath, _ := filepath.Abs("../config/kubernetes/default/deployments/kubevalidator.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations, err := context.AnnotateFile(&fileContents, &github.CommitFile{
		Filename: github.String("config/kubernetes/default/deployments/kubevalidator.yaml"),
	})
	if err != nil {
		t.Errorf("AnnotateFile failed with %s", err)
	}

	var want []*github.CheckRunAnnotation

	if !reflect.DeepEqual(checkRunAnnotations, want) {
		t.Errorf("context.AnnotateFile returned %+v, want %+v", checkRunAnnotations, want)
	}
}

func TestAnnotationsForInvalidFile(t *testing.T) {
	context := &Context{}
	filePath, _ := filepath.Abs("../fixtures/invalid.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations, err := context.AnnotateFile(&fileContents, &github.CommitFile{
		BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		Filename: github.String("deployment.yaml"),
	})
	if err != nil {
		t.Errorf("AnnotateFile failed with %s", err)
	}
	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("template is required"),
		Message:      github.String("template: template is required"),
		RawDetails:   github.String("asdf"),
	}, {
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("Invalid type. Expected: integer, given: string"),
		Message:      github.String("spec.replicas: Invalid type. Expected: integer, given: string"),
		RawDetails:   github.String("asdf"),
	}}

	if len(checkRunAnnotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(checkRunAnnotations), len(want))
	}

	for i, annotation := range checkRunAnnotations {
		if !reflect.DeepEqual(annotation.Title, want[i].Title) {
			t.Errorf("[%d]title was %+v, want %+v", i, *annotation.Title, *want[i].Title)
		}

		if !reflect.DeepEqual(annotation.Message, want[i].Message) {
			t.Errorf("[%d]message was %+v, want %+v", i, *annotation.Message, *want[i].Message)
		}

	}
}
