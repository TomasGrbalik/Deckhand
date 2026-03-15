package service

import (
	"fmt"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// ConfigWriter persists project config to disk.
type ConfigWriter interface {
	Save(path string, proj *domain.Project) error
}

// EnvironmentRecreator re-renders templates and recreates containers.
type EnvironmentRecreator interface {
	Up(build bool) error
}

// PortService manages port mappings on a project. Add and Remove handle the
// full orchestration: validate → modify → persist config → recreate containers.
type PortService struct {
	project    *domain.Project
	configPath string
	config     ConfigWriter
	env        EnvironmentRecreator
}

// NewPortService creates a PortService. The configPath, config writer, and
// environment recreator are only needed for Add/Remove — List works without them.
func NewPortService(
	project *domain.Project,
	configPath string,
	config ConfigWriter,
	env EnvironmentRecreator,
) *PortService {
	return &PortService{
		project:    project,
		configPath: configPath,
		config:     config,
		env:        env,
	}
}

// List returns the current port mappings.
func (s *PortService) List() []domain.PortMapping {
	return s.project.Ports
}

// Add validates, adds a port mapping, persists config, and recreates containers.
func (s *PortService) Add(port int, name, protocol string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of range (1-65535)", port)
	}

	if protocol != "http" && protocol != "tcp" {
		return fmt.Errorf("invalid protocol %q (must be http or tcp)", protocol)
	}

	for _, p := range s.project.Ports {
		if p.Port == port {
			return fmt.Errorf("port %d already mapped", port)
		}
	}

	s.project.Ports = append(s.project.Ports, domain.PortMapping{
		Port:     port,
		Name:     name,
		Protocol: protocol,
	})

	return s.persistAndRecreate()
}

// Remove removes a port mapping, persists config, and recreates containers.
func (s *PortService) Remove(port int) error {
	found := false
	for i, p := range s.project.Ports {
		if p.Port == port {
			s.project.Ports = append(s.project.Ports[:i], s.project.Ports[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("port %d not found", port)
	}

	return s.persistAndRecreate()
}

// persistAndRecreate saves config to disk and recreates the environment.
func (s *PortService) persistAndRecreate() error {
	if err := s.config.Save(s.configPath, s.project); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if err := s.env.Up(false); err != nil {
		return fmt.Errorf("recreating environment: %w", err)
	}

	return nil
}
