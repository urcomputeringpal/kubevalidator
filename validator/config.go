package validator

import (
	"fmt"
	"path"
	"regexp"

	"github.com/google/go-github/github"
)

// KubeValidatorConfig maps globs of Kubernetes config to schemas which validate
// them.
type KubeValidatorConfig struct {
	APIVersion string                   `yaml:"apiversion"`
	Kind       string                   `yaml:"kind"`
	Spec       *KubeValidatorConfigSpec `yaml:"spec"`
}

// KubeValidatorConfigSpec contains a list of manifests
type KubeValidatorConfigSpec struct {
	Manifests []*KubeValidatorConfigManifest `yaml:"manifests"`
}

// KubeValidatorConfigManifest contains a glob and a list of schema
type KubeValidatorConfigManifest struct {
	Glob    string                       `yaml:"glob"`
	Schemas []*KubeValidatorConfigSchema `yaml:"schemas,omitempty"`
}

// KubeValidatorConfigSchema contains options for kubeval
type KubeValidatorConfigSchema struct {
	Name       string `yaml:"name,omitempty"`
	SchemaFork string `yaml:"schemaFork,omitempty"`

	Version     string `yaml:"version,omitempty"`
	ConfigType  string `yaml:"type,omitempty"`
	Strict      bool   `yaml:"strict,omitempty"`
	LineNumbers bool   `yaml:"lineNumbers,omitempty"`
}

func (config *KubeValidatorConfig) matchingCandidates(context *Context, files []*github.CommitFile) []*Candidate {
	var candidates []*Candidate

	for _, file := range files {
		if config.Spec != nil {
			spec := *config.Spec
			for _, manifestConfig := range spec.Manifests {
				if matched, _ := path.Match(manifestConfig.Glob, file.GetFilename()); matched {
					candidate := NewCandidate(context, file, manifestConfig.Schemas)
					candidates = append(candidates, candidate)
				}
			}
		}
	}

	return candidates
}

// Valid returns a boolean indicatating whether or not the config is well formed
// TODO replace me with an actual schema
func (config *KubeValidatorConfig) Valid() bool {
	re := regexp.MustCompile(`(?mi)^[a-z][a-z\-]{0,38}$`)
	if config.Spec != nil {
		spec := *config.Spec
		for _, manifest := range spec.Manifests {
			for _, schema := range manifest.Schemas {
				if schema.SchemaFork != "" && !re.MatchString(schema.SchemaFork) {
					return false
				}
			}
		}
	}
	return true
}

// SchemaLocation composes SchemaFork with a base url
func (schema *KubeValidatorConfigSchema) SchemaLocation() string {
	schemaFork := schema.SchemaFork
	if schemaFork == "" {
		schemaFork = "garethr"
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s", schemaFork)
}
