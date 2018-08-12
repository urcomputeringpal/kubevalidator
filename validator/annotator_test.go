package validator

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
)

func TestAnnotationsForValidFile(t *testing.T) {
	filePath, _ := filepath.Abs("../config/kubernetes/default/deployments/kubevalidator.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations, err := AnnotateFile(&fileContents, &github.CommitFile{
		Filename: github.String("config/kubernetes/default/deployments/kubevalidator.yaml"),
	})
	if err != nil {
		t.Errorf("AnnotateFile failed with %s", err)
	}

	var want []*github.CheckRunAnnotation

	if !reflect.DeepEqual(checkRunAnnotations, want) {
		t.Errorf("AnnotateFile returned %+v, want %+v", checkRunAnnotations, want)
	}
}

func TestAnnotationsForInvalidFile(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/invalid.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations, err := AnnotateFile(&fileContents, &github.CommitFile{
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
		Title:        github.String("template: template is required"),
		Message:      github.String("map[property:template field:template context:(root).spec]"),
		RawDetails:   github.String("asdf"),
	}, {
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("spec.replicas: Invalid type. Expected: integer, given: string"),
		Message:      github.String("map[expected:integer given:string field:spec.replicas context:(root).spec.replicas]"),
		RawDetails:   github.String("asdf"),
	}}

	if len(checkRunAnnotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(checkRunAnnotations), len(want))
	}

	for i, annotation := range checkRunAnnotations {
		if !reflect.DeepEqual(annotation.Title, want[i].Title) {
			t.Errorf("[%d]title was %+v, want %+v", i, *annotation.Title, *want[i].Title)
		}

	}
}

func TestAnnotationsWithCustomSchemaSuccess(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/invalid/1.6.0/volumeerror.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations, err := AnnotateFileWithSchema(&fileContents,
		&github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
			Filename: github.String("volumeerror.yaml"),
		},
		&KubeValidatorConfigSchema{
			version: "1.10.0",
		})
	if err != nil {
		t.Errorf("AnnotateFile failed with %s", err)
	}

	if len(checkRunAnnotations) != 0 {
		t.Errorf("%d annotations returned, expected 0: %+v", len(checkRunAnnotations), checkRunAnnotations[0].GetTitle())
	}
}

func TestAnnotationsWithCustomSchemaFailure(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/invalid/1.6.0/volumeerror.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	checkRunAnnotations, err := AnnotateFileWithSchema(&fileContents,
		&github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volumeerror.yaml"),
			Filename: github.String("volumeerror.yaml"),
		},
		&KubeValidatorConfigSchema{
			version: "1.6.0",
		})
	if err != nil {
		t.Errorf("AnnotateFile failed with %s", err)
	}
	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/volume_attachment_list.yaml"),
		StartLine:    github.Int(1),
		EndLine:      github.Int(1),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating VolumeError"),
		Message:      github.String("Schema file not found! This likely means this type isn't available in configured schema"),
		RawDetails:   github.String("asdf"),
	}}

	if len(checkRunAnnotations) != len(want) {
		t.Errorf("a total of %d annotations were returned, wanted %d", len(checkRunAnnotations), len(want))
	}

	for i, annotation := range checkRunAnnotations {
		if !reflect.DeepEqual(annotation.Title, want[i].Title) {
			t.Errorf("[%d]title was %+v, want %+v", i, *annotation.Title, *want[i].Title)
		}

	}
}
