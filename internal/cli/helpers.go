package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TomasGrbalik/deckhand/internal/config"
	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
	"github.com/TomasGrbalik/deckhand/internal/infra/template"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

// projectDir returns the current working directory.
func projectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return dir, nil
}

// loadProject loads the project config from the current directory.
func loadProject(dir string) (*domain.Project, error) {
	cfgPath := config.ProjectConfigPath(dir)
	proj, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return proj, nil
}

// newEnvironmentService creates an EnvironmentService wired to real infra.
func newEnvironmentService(proj domain.Project, dir string) *service.EnvironmentService {
	return service.NewEnvironmentService(
		&template.EmbeddedSource{},
		docker.NewCompose(),
		proj,
		dir,
	)
}

// newContainerService creates a ContainerService wired to a real Docker client.
func newContainerService() (*service.ContainerService, func(), error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to docker: %w", err)
	}
	cleanup := func() { _ = client.Close() }
	return service.NewContainerService(docker.NewContainer(client.API())), cleanup, nil
}

// newStatusService creates a StatusService wired to a real Docker client.
func newStatusService() (*service.StatusService, func(), error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to docker: %w", err)
	}
	cleanup := func() { _ = client.Close() }
	return service.NewStatusService(docker.NewContainer(client.API())), cleanup, nil
}

// dirName returns the base name of the given directory path.
func dirName(dir string) string {
	return filepath.Base(dir)
}
