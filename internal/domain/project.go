package domain

// Project represents the user's project configuration.
// This is what .deckhand.yaml deserializes into.
type Project struct {
	Name     string            `yaml:"project"`
	Template string            `yaml:"template"`
	Ports    []PortMapping     `yaml:"ports"`
	Env      map[string]string `yaml:"env"`
}
