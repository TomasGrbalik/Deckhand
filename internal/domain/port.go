package domain

// PortMapping represents a single port exposed by the dev environment.
type PortMapping struct {
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
	Internal bool   `yaml:"internal"`
}
