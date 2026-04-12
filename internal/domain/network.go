package domain

// NetworkConfig holds the shared Docker network settings for SSH access.
// When configured in global config, devcontainers get a static IP on
// this network for direct SSH access via Tailscale subnet routing.
type NetworkConfig struct {
	Name    string `yaml:"name"`
	Subnet  string `yaml:"subnet"`
	Gateway string `yaml:"gateway"`
}

// IsConfigured returns true if all required network fields are set.
func (n NetworkConfig) IsConfigured() bool {
	return n.Name != "" && n.Subnet != "" && n.Gateway != ""
}
