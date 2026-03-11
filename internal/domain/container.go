package domain

// Container represents a running (or stopped) container managed by deckhand.
type Container struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Service string `yaml:"service"`
	Status  string `yaml:"status"`
	Health  string `yaml:"health"`
}
