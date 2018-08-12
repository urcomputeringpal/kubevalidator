package validator

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestUnmarshalConfig(t *testing.T) {
	filePath, _ := filepath.Abs("../.github/kubevalidator.yaml")
	fileContents, _ := ioutil.ReadFile(filePath)
	config := &KubeValidatorConfig{}
	configBytes := []byte(fileContents)
	err := yaml.Unmarshal(configBytes, config)
	if err != nil {
		t.Errorf("Unmarshaling kubevalidator.yaml failed with %v", err)
	}
}
