package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	goyaml "gopkg.in/yaml.v3"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// Load reads a .deckhand.yaml file at path and returns the parsed Project.
func Load(path string) (*domain.Project, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file not found %s: %w", path, err)
	}

	k := koanf.New(".")

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	var proj domain.Project
	// Use "yaml" struct tags so domain types don't need koanf-specific tags.
	if err := k.UnmarshalWithConf("", &proj, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return nil, fmt.Errorf("unmarshalling config %s: %w", path, err)
	}

	// Treat missing version as version 1 (backward compatible).
	if proj.Version == 0 {
		proj.Version = 1
	}

	if proj.Version > 1 {
		return nil, fmt.Errorf("config version %d is not supported — please upgrade deckhand", proj.Version)
	}

	return &proj, nil
}

// Save writes a Project back to a .deckhand.yaml file at path.
func Save(path string, proj *domain.Project) error {
	// Ensure version is always written.
	if proj.Version == 0 {
		proj.Version = 1
	}

	data, err := goyaml.Marshal(proj)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // project config is non-secret, should be readable by team members
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}

// LoadGlobal reads the global config file and returns the parsed GlobalConfig.
// If the file does not exist, it returns a zero-value GlobalConfig and nil error.
// If the file exists but contains invalid YAML, it returns an error.
func LoadGlobal(path string) (*domain.GlobalConfig, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &domain.GlobalConfig{}, nil
		}
		return nil, fmt.Errorf("checking global config %s: %w", path, err)
	}

	k := koanf.New(".")

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("parsing global config %s: %w", path, err)
	}

	var cfg domain.GlobalConfig
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return nil, fmt.Errorf("unmarshalling global config %s: %w", path, err)
	}

	return &cfg, nil
}
