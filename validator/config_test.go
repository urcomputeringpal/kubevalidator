package validator

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-github/github"
	yaml "gopkg.in/yaml.v2"
)

func TestConfigMatchesCandidates(t *testing.T) {
	filePath, _ := filepath.Abs("../fixtures/kubevalidator.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	config := &KubeValidatorConfig{}
	configBytes := []byte(fileContents)
	err := yaml.Unmarshal(configBytes, config)
	if err != nil {
		t.Errorf("Unmarshaling kubevalidator.yaml failed with %v", err)
		return
	}

	var files []*github.CommitFile
	files = append(files, &github.CommitFile{
		Filename: github.String("fixtures/deployment.yaml"),
	})
	files = append(files, &github.CommitFile{
		Filename: github.String("README.md"),
	})
	candidates := config.matchingCandidates(files)
	if len(candidates) != 1 {
		t.Errorf("Expected 1 match, got %d", len(candidates))
	}
}

func TestEmptyConfigMatchesNothing(t *testing.T) {
	config := &KubeValidatorConfig{}
	var files []*github.CommitFile
	file := &github.CommitFile{
		Filename: github.String("important.yaml"),
	}
	files = append(files, file)
	candidates := config.matchingCandidates(files)
	if len(candidates) != 0 {
		t.Errorf("found unexpected candidates! %v", candidates)
	}
}
