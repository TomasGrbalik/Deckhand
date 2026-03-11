package config

import "path/filepath"

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
