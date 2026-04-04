package domain

// CompanionService describes a companion service available in the registry.
type CompanionService struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Image       string            `yaml:"image"`
	Ports       []int             `yaml:"ports"`
	Environment map[string]string `yaml:"environment"`
	HealthCheck HealthCheck       `yaml:"healthcheck"`
	Volumes     []string          `yaml:"volumes"`
}

// HealthCheck defines how to verify a companion service is ready.
type HealthCheck struct {
	Test     string `yaml:"test"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}

// ServiceConfig represents a companion service selection in the project config.
type ServiceConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version,omitempty"`
	Enabled bool   `yaml:"enabled"`
}
