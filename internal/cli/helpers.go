package cli

import (
	"errors"
	"fmt"
	"io/fs"
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
		nil, // no volume manager — commands that need it create one explicitly
		proj,
		dir,
	)
}

// newEnvironmentServiceWithVolumes creates an EnvironmentService with volume
// management support. Used by destroy which needs to discover and remove
// labeled volumes.
func newEnvironmentServiceWithVolumes(proj domain.Project, dir string) (*service.EnvironmentService, func(), error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to docker: %w", err)
	}
	cleanup := func() { _ = client.Close() }

	volMgr := newVolumeListerAdapter(docker.NewVolume(client.API()))
	svc := service.NewEnvironmentService(
		&template.EmbeddedSource{},
		docker.NewCompose(),
		volMgr,
		proj,
		dir,
	)
	return svc, cleanup, nil
}

// volumeListerAdapter adapts docker.Volume (infra) to service.VolumeManager
// by mapping docker.VolumeInfo to service.VolumeInfo.
type volumeListerAdapter struct {
	vol *docker.Volume
}

func newVolumeListerAdapter(vol *docker.Volume) *volumeListerAdapter {
	return &volumeListerAdapter{vol: vol}
}

// ListByProject implements service.VolumeManager.
func (a *volumeListerAdapter) ListByProject(projectName string) ([]service.VolumeInfo, error) {
	infos, err := a.vol.ListByProject(projectName)
	if err != nil {
		return nil, err
	}
	result := make([]service.VolumeInfo, len(infos))
	for i, info := range infos {
		result[i] = service.VolumeInfo{Name: info.Name}
	}
	return result, nil
}

// Remove implements service.VolumeManager.
func (a *volumeListerAdapter) Remove(volumeName string) error {
	return a.vol.Remove(volumeName)
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
	adapter := &containerListerAdapter{ctr: docker.NewContainer(client.API())}
	return service.NewStatusService(adapter), cleanup, nil
}

// containerListerAdapter adapts docker.Container (infra) to service.ContainerLister
// by mapping docker.ContainerInfo to domain.Container. This keeps infra free of
// domain imports.
type containerListerAdapter struct {
	ctr *docker.Container
}

// ListByProject implements service.ContainerLister.
func (a *containerListerAdapter) ListByProject(projectName string) ([]domain.Container, error) {
	infos, err := a.ctr.ListByProject(projectName)
	if err != nil {
		return nil, err
	}
	return mapContainerInfos(infos), nil
}

// ListAll implements service.ContainerLister.
func (a *containerListerAdapter) ListAll() ([]domain.Container, error) {
	infos, err := a.ctr.ListAll()
	if err != nil {
		return nil, err
	}
	return mapContainerInfos(infos), nil
}

func mapContainerInfos(infos []docker.ContainerInfo) []domain.Container {
	result := make([]domain.Container, len(infos))
	for i, info := range infos {
		ports := make([]domain.ContainerPort, len(info.Ports))
		for j, p := range info.Ports {
			ports[j] = domain.ContainerPort{Public: p.Public, Private: p.Private}
		}
		result[i] = domain.Container{
			ID:      info.ID,
			Name:    info.Name,
			Service: info.Service,
			Project: info.Project,
			Image:   info.Image,
			State:   info.State,
			Status:  info.Status,
			Created: info.Created,
			Ports:   ports,
		}
	}
	return result
}

// dirName returns the base name of the given directory path.
func dirName(dir string) string {
	return filepath.Base(dir)
}

// compositeSource tries multiple TemplateSource implementations in order.
// The first source that succeeds wins. This lets user templates on disk
// override embedded templates for Load/LoadMeta operations.
type compositeSource struct {
	sources []service.TemplateSource
}

// Load implements service.TemplateSource. It only falls back to the next source
// when the error indicates the template was not found (fs.ErrNotExist). Real
// read/parse errors are returned immediately so broken overrides surface.
func (c *compositeSource) Load(name string) (string, string, error) {
	var lastErr error
	for _, src := range c.sources {
		df, comp, err := src.Load(name)
		if err == nil {
			return df, comp, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", "", err
		}
		lastErr = err
	}
	return "", "", lastErr
}

// LoadMeta implements service.TemplateSource. Same fallback semantics as Load.
func (c *compositeSource) LoadMeta(name string) (*domain.TemplateMeta, error) {
	var lastErr error
	for _, src := range c.sources {
		meta, err := src.LoadMeta(name)
		if err == nil {
			return meta, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}
