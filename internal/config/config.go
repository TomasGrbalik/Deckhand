package config

import (
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

	return &proj, nil
}

// Save writes a Project back to a .deckhand.yaml file at path.
func Save(path string, proj *domain.Project) error {
	data, err := goyaml.Marshal(proj)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // project config is non-secret, should be readable by team members
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}
