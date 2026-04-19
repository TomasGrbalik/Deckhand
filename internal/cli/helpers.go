package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// stderrLogger is a logger that writes to stderr.
func stderrLogger(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// newEnvironmentService creates an EnvironmentService wired to real infra.
// It loads the global config for mount merging during Up. If the global
// config has a network block, it also wires up a Docker network checker
// and the network state path.
func newEnvironmentService(proj domain.Project, dir string) (*service.EnvironmentService, func(), error) {
	globalCfg, err := loadGlobalConfig()
	if err != nil {
		return nil, nil, err
	}
	svc := service.NewEnvironmentService(
		templateSourceForProject(dir),
		docker.NewCompose(),
		nil, // no volume manager — commands that need it create one explicitly
		globalCfg,
		proj,
		dir,
	)
	svc.SetLogger(stderrLogger)

	cleanup := func() {}
	if globalCfg.Network.IsConfigured() {
		statePath, pathErr := config.NetworkStatePath()
		if pathErr != nil {
			return nil, nil, fmt.Errorf("resolving network state path: %w", pathErr)
		}
		client, clientErr := docker.NewClient(context.Background())
		if clientErr != nil {
			return nil, nil, fmt.Errorf("connecting to docker: %w", clientErr)
		}
		cleanup = func() { _ = client.Close() }
		svc.SetNetworkSupport(docker.NewNetwork(client.API()), statePath)
	}

	return svc, cleanup, nil
}

// newEnvironmentServiceForDown creates a lightweight EnvironmentService for
// down/status commands that don't need global config or mount merging.
func newEnvironmentServiceForDown(proj domain.Project, dir string) *service.EnvironmentService {
	return service.NewEnvironmentService(
		templateSourceForProject(dir),
		docker.NewCompose(),
		nil,
		domain.GlobalConfig{},
		proj,
		dir,
	)
}

// loadGlobalConfig loads the global config. Returns an empty config if the
// file doesn't exist or the config dir can't be resolved. Returns an error
// for real failures (e.g., malformed YAML).
func loadGlobalConfig() (domain.GlobalConfig, error) {
	path, pathErr := config.GlobalConfigPath()
	if pathErr != nil {
		return domain.GlobalConfig{}, nil //nolint:nilerr // path resolution non-fatal (e.g., no home dir)
	}
	cfg, err := config.LoadGlobal(path)
	if err != nil {
		return domain.GlobalConfig{}, fmt.Errorf("loading global config: %w", err)
	}
	return *cfg, nil
}

// newEnvironmentServiceWithVolumes creates an EnvironmentService with volume
// management support. Used by destroy which needs to discover and remove
// labeled volumes and free network IPs.
func newEnvironmentServiceWithVolumes(proj domain.Project, dir string) (*service.EnvironmentService, func(), error) {
	globalCfg, err := loadGlobalConfig()
	if err != nil {
		return nil, nil, err
	}

	client, err := docker.NewClient(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to docker: %w", err)
	}
	cleanup := func() { _ = client.Close() }

	volMgr := newVolumeListerAdapter(docker.NewVolume(client.API()))
	svc := service.NewEnvironmentService(
		templateSourceForProject(dir),
		docker.NewCompose(),
		volMgr,
		globalCfg,
		proj,
		dir,
	)
	svc.SetLogger(stderrLogger)

	if globalCfg.Network.IsConfigured() {
		statePath, pathErr := config.NetworkStatePath()
		if pathErr != nil {
			cleanup()
			return nil, nil, fmt.Errorf("resolving network state path: %w", pathErr)
		}
		svc.SetNetworkSupport(nil, statePath) // No checker needed for destroy.
	}

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
		result[i] = service.VolumeInfo{Name: info.Name, Size: info.Size}
	}
	return result, nil
}

// Remove implements service.VolumeManager.
func (a *volumeListerAdapter) Remove(volumeName string) error {
	return a.vol.Remove(volumeName)
}

// resolveExecUser looks up the ExecUser declared in the project's template
// metadata. Returns empty string when the project has no template set or when
// the metadata doesn't declare exec_user. The template name defaults to "base"
// matching TemplateService.Render to keep behavior consistent.
func resolveExecUser(projectDir, templateName string) (string, error) {
	name := templateName
	if name == "" {
		name = "base"
	}
	meta, err := templateSourceForProject(projectDir).LoadMeta(name)
	if err != nil {
		return "", fmt.Errorf("loading template metadata for %q: %w", name, err)
	}
	return meta.ExecUser, nil
}

// newContainerService creates a ContainerService wired to a real Docker client.
func newContainerService() (*service.ContainerService, func(), error) {
	client, err := docker.NewClient(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to docker: %w", err)
	}
	cleanup := func() { _ = client.Close() }
	return service.NewContainerService(docker.NewContainer(client.API())), cleanup, nil
}

// newStatusService creates a StatusService wired to a real Docker client.
func newStatusService() (*service.StatusService, func(), error) {
	client, err := docker.NewClient(context.Background())
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

// localTemplateSource returns a FilesystemSource for project-local templates
// in .deckhand/templates/ under the given project directory.
func localTemplateSource(projectDir string) *template.FilesystemSource {
	return &template.FilesystemSource{
		Dir:         filepath.Join(projectDir, ".deckhand", "templates"),
		SourceLabel: "local",
	}
}

// userTemplateSource returns a FilesystemSource for user-global templates
// in ~/.config/deckhand/templates/. Returns nil if the home directory cannot
// be resolved.
func userTemplateSource() *template.FilesystemSource {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return &template.FilesystemSource{
		Dir: filepath.Join(home, ".config", "deckhand", "templates"),
	}
}

// templateSourceForProject builds a compositeSource that tries local templates
// first, then user-global, then embedded. This gives local-first precedence
// for Load/LoadMeta operations.
func templateSourceForProject(projectDir string) service.TemplateSource {
	embedded := &template.EmbeddedSource{}
	local := localTemplateSource(projectDir)
	sources := []service.TemplateSource{local}
	if user := userTemplateSource(); user != nil {
		sources = append(sources, user)
	}
	sources = append(sources, embedded)
	return &compositeSource{sources: sources}
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

// displayGlobalMountSummary prints an informational summary of active global
// mounts. If no global config exists or it has no mounts, nothing is printed.
// Config load errors are printed as warnings rather than failing init.
func displayGlobalMountSummary(w io.Writer) {
	cfg, err := loadGlobalConfig()
	if err != nil {
		fmt.Fprintf(w, "warning: %s\n", err)
		return
	}

	var entries []string
	for _, v := range cfg.Mounts.Volumes {
		entries = append(entries, v.Name+" (volume)")
	}
	for _, s := range cfg.Mounts.Secrets {
		entries = append(entries, s.Name+" (secret)")
	}
	for _, s := range cfg.Mounts.Sockets {
		entries = append(entries, s.Name+" (socket)")
	}

	if len(entries) == 0 {
		return
	}

	fmt.Fprintln(w, "\nGlobal mounts:")
	for _, e := range entries {
		fmt.Fprintf(w, "  %s\n", e)
	}
}
