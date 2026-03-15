package domain

import "time"

// ContainerPort represents a port mapping on a running container.
type ContainerPort struct {
	Public  int
	Private int
}

// Container represents a running (or stopped) container managed by deckhand.
type Container struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Service string `yaml:"service"`
	Project string `yaml:"project"`
	Image   string `yaml:"image"`
	Status  string `yaml:"status"`
	State   string `yaml:"state"`
	Health  string `yaml:"health"`
	Created time.Time
	Ports   []ContainerPort
}
