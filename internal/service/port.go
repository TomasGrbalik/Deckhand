package service

import (
	"fmt"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// PortService manages port mappings on a project.
type PortService struct {
	project *domain.Project
}

// NewPortService creates a PortService for the given project.
func NewPortService(project *domain.Project) *PortService {
	return &PortService{project: project}
}

// List returns the current port mappings.
func (s *PortService) List() []domain.PortMapping {
	return s.project.Ports
}

// Add adds a port mapping. Returns an error if the port is already mapped,
// out of range, or the protocol is invalid.
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

	return nil
}

// Remove removes a port mapping. Returns an error if the port is not found.
func (s *PortService) Remove(port int) error {
	for i, p := range s.project.Ports {
		if p.Port == port {
			s.project.Ports = append(s.project.Ports[:i], s.project.Ports[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("port %d not found", port)
}
