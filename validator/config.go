package validator

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
	Version    string `yaml:"version,omitempty"`
	BaseURL    string `yaml:"baseURL,omitempty"`
	ConfigType string `yaml:"type,omitempty"`
	Strict     bool   `yaml:"strict,omitempty"`
}
