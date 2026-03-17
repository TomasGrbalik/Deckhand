package domain

// Project represents the user's project configuration.
// This is what .deckhand.yaml deserializes into.
type Project struct {
	Name      string            `yaml:"project"`
	Template  string            `yaml:"template"`
	Ports     []PortMapping     `yaml:"ports"`
	Env       map[string]string `yaml:"env"`
	Variables map[string]string `yaml:"variables,omitempty"`
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
}
