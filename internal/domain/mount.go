package domain

import "fmt"

// VolumeMount represents a named Docker volume.
type VolumeMount struct {
	Name    string `yaml:"name"`
	Target  string `yaml:"target"`
	Enabled *bool  `yaml:"enabled,omitempty"`
}

// SecretMount represents a credential injected as an env var, file, or both.
// The source field holds a raw string — env var references (${VAR}) and file
// paths (~/.gitconfig) are resolved by the service layer, not here.
type SecretMount struct {
	Name     string `yaml:"name"`
	Source   string `yaml:"source,omitempty"`
	Target   string `yaml:"target,omitempty"`
	Env      string `yaml:"env,omitempty"`
	ReadOnly bool   `yaml:"readonly,omitempty"`
	Enabled  *bool  `yaml:"enabled,omitempty"`
}

// Validate checks that a SecretMount has at least one output (env or target).
func (s SecretMount) Validate() error {
	if s.Env == "" && s.Target == "" {
		return fmt.Errorf("secret %q must have at least one of env or target", s.Name)
	}
	return nil
}

// SocketMount represents a Unix socket forwarded from host into the container.
type SocketMount struct {
	Name    string `yaml:"name"`
	Source  string `yaml:"source,omitempty"`
	Target  string `yaml:"target,omitempty"`
	Env     string `yaml:"env,omitempty"`
	Enabled *bool  `yaml:"enabled,omitempty"`
}

// Mounts groups the three mount primitive slices.
type Mounts struct {
	Volumes []VolumeMount `yaml:"volumes,omitempty"`
	Secrets []SecretMount `yaml:"secrets,omitempty"`
	Sockets []SocketMount `yaml:"sockets,omitempty"`
}
