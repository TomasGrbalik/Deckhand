package domain

// GlobalConfig represents user-wide preferences loaded from
// ~/.config/deckhand/config.yaml.
type GlobalConfig struct {
	Defaults GlobalDefaults `yaml:"defaults"`
	SSH      SSHConfig      `yaml:"ssh"`
	Mounts   Mounts         `yaml:"mounts"`
}

// GlobalDefaults holds default settings that apply to all projects
// unless overridden by project config.
type GlobalDefaults struct {
	Template string `yaml:"template"`
	Shell    string `yaml:"shell"`
}

// SSHConfig holds SSH connection settings for the connect command.
type SSHConfig struct {
	User string `yaml:"user"`
	Host string `yaml:"host"`
}
