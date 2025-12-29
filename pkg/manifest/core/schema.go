package manifestcore

type Manifest struct {
	Version    string                    `yaml:"version"`
	Inputs     map[string]Input          `yaml:"inputs,omitempty"`
	Metadata   Metadata                  `yaml:"metadata"`
	Targets    map[string]TargetManifest `yaml:"targets,omitempty"`
	Components []Component               `yaml:"components"`
}

type Input struct {
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Default     string `yaml:"default,omitempty"`
	Sensitive   bool   `yaml:"sensitive,omitempty"`
}

type Metadata struct {
	ID          string            `yaml:"id"`
	Description string            `yaml:"description"`
	Owner       Owner             `yaml:"owner"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

type Owner struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type TargetManifest struct {
	Type   string
	Config map[string]interface{} `yaml:"config"`
}

type ComponentType string

type Hooks struct {
	Create struct {
		PreRun  []string `yaml:"preRun,omitempty"`
		PostRun []string `yaml:"postRun,omitempty"`
	} `yaml:"create,omitempty"`
	Destroy struct {
		PreRun  []string `yaml:"preRun,omitempty"`
		PostRun []string `yaml:"postRun,omitempty"`
	} `yaml:"destroy,omitempty"`
}

type ComponentSource struct {
	Content string `yaml:"content,omitempty"`
	Path    string `yaml:"path,omitempty"`
}

type Component struct {
	Name      string            `yaml:"name"`
	Type      string            `yaml:"type"`
	DependsOn []string          `yaml:"dependsOn,omitempty"`
	Config    map[string]string `yaml:"config,omitempty"`
	Hooks     Hooks             `yaml:"hooks,omitempty"`
	Source    ComponentSource   `yaml:"source,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Outputs   []string          `yaml:"outputs,omitempty"`
	Target    string            `yaml:"target,omitempty"`

	LoadedConfig interface{} `yaml:"-"`
}
