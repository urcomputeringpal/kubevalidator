package validator

// KubeValidatorConfig maps globs of Kubernetes config to schemas which validate
// them.
type KubeValidatorConfig struct {
	apiversion string                   `yaml:"apiversion"`
	kind       string                   `yaml:"kind"`
	spec       *KubeValidatorConfigSpec `yaml:"spec"`
}

// KubeValidatorConfigSpec contains a list of manifests
type KubeValidatorConfigSpec struct {
	manifests []*KubeValidatorConfigManifest `yaml:"manifests"`
}

// KubeValidatorConfigManifest contains a glob and a list of schema
type KubeValidatorConfigManifest struct {
	glob    string                       `yaml:"glob"`
	schemas []*KubeValidatorConfigSchema `yaml:"schemas,omitempty"`
}

// KubeValidatorConfigSchema contains options for kubeval
type KubeValidatorConfigSchema struct {
	version    string `yaml:"version"`
	baseURL    string `yaml:"baseURL,omitempty"`
	configType string `yaml:"type,omitempty"`
	strict     bool   `yaml:"strict,omitempty"`
}
