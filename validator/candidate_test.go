package validator

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
)

const (
	// baseURLPath is a non-empty Client.BaseURL path to use during tests,
	// to ensure relative URLs are used for all endpoints. See issue #752.
	baseURLPath = "/api-v3"
)

// setup sets up a test HTTP server along with a github.Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup() (client *github.Client, mux *http.ServeMux, serverURL string, teardown func()) {
	// mux is the HTTP request multiplexer used with the test server.
	mux = http.NewServeMux()

	// We want to ensure that tests catch mistakes where the endpoint URL is
	// specified as absolute rather than relative. It only makes a difference
	// when there's a non-empty base URL path. So, use that. See issue #752.
	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		fmt.Fprintln(os.Stderr, "\tSee https://github.com/google/go-github/issues/752 for information.")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	client = github.NewClient(nil)
	url, _ := url.Parse(server.URL + baseURLPath + "/")
	client.BaseURL = url
	client.UploadURL = url

	return client, mux, server.URL, server.Close
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

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

func TestAnnotationsForInvalidArrayCandidate(t *testing.T) {
	candidate := NewCandidate(
		&Context{
			Event: &github.CheckSuiteEvent{},
		}, &github.CommitFile{
			BlobURL:  github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
			Filename: github.String("deployment.yaml"),
		}, nil)

	filePath, _ := filepath.Abs("../fixtures/invalid/deployment/extra-field-in-containers.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	candidate.setBytes(&fileContents)
	annotations := candidate.Validate()

	want := []*github.CheckRunAnnotation{{
		FileName:     github.String("deployment.yaml"),
		BlobHRef:     github.String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/deployment.yaml"),
		StartLine:    github.Int(16),
		EndLine:      github.Int(28),
		WarningLevel: github.String("failure"),
		Title:        github.String("Error validating Deployment against master schema"),
		Message:      github.String("spec.template.spec.containers.0.name: Invalid type. Expected: string, given: integer"),
		RawDetails:   github.String("* context: (root).spec.template.spec.containers.0.name\n* expected: string\n* field: spec.template.spec.containers.0.name\n* given: integer\n"),
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
