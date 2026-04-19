package domain

// Project represents the user's project configuration.
// This is what .deckhand.yaml deserializes into.
type Project struct {
	Version   int               `yaml:"version"`
	Name      string            `yaml:"project"`
	Template  string            `yaml:"template"`
	Ports     []PortMapping     `yaml:"ports"`
	Env       map[string]string `yaml:"env"`
	Variables map[string]string `yaml:"variables,omitempty"`
	Mounts    Mounts            `yaml:"mounts,omitempty"`
	Services  []ServiceConfig   `yaml:"services,omitempty"`
}

// TemplateVariable describes a single configurable variable in a template.
type TemplateVariable struct {
	Default     string `yaml:"default"`
	Description string `yaml:"description"`
}

// TemplateMeta holds metadata parsed from a template's metadata.yaml.
type TemplateMeta struct {
	Name        string                      `yaml:"name"`
	Description string                      `yaml:"description"`
	Variables   map[string]TemplateVariable `yaml:"variables"`
	Mounts      Mounts                      `yaml:"mounts,omitempty"`
	// Command overrides the devcontainer's long-running command in the
	// rendered compose file. When empty, the service layer defaults to
	// "sleep infinity".
	Command string `yaml:"command,omitempty"`
	// ExecUser is the user that `deckhand shell` and `deckhand exec` drop
	// into, independent of the Dockerfile's USER directive. Empty means the
	// image's default user is used.
	ExecUser string `yaml:"exec_user,omitempty"`
}

// TemplateInfo describes a discovered template for listing purposes.
type TemplateInfo struct {
	Name        string
	Description string
	Source      string // "builtin", "user", or "local"
}
