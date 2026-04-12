package config

import (
	"os"
	"path/filepath"
)

// ProjectConfigPath returns the path to .deckhand.yaml relative to the given
// project directory.
func ProjectConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".deckhand.yaml")
}

// GeneratedDir returns the path to the .deckhand/ directory where generated
// files (Dockerfile, docker-compose.yml) are written.
func GeneratedDir(projectDir string) string {
	return filepath.Join(projectDir, ".deckhand")
}

// GlobalConfigPath returns the path to the global deckhand config file.
// It uses os.UserConfigDir() for the platform-appropriate base path
// (e.g. ~/.config on Linux).
func GlobalConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "deckhand", "config.yaml"), nil
}

// NetworkStatePath returns the path to the network state file that tracks
// project-to-IP assignments. Returns empty string if the config dir can't
// be resolved.
func NetworkStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "deckhand", "network-state.yaml"), nil
}
