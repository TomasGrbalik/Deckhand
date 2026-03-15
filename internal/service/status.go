package service

import (
	"fmt"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// ContainerLister queries for deckhand-managed containers.
// The infra layer provides the raw data; an adapter maps it to domain types.
type ContainerLister interface {
	ListByProject(projectName string) ([]domain.Container, error)
	ListAll() ([]domain.Container, error)
}

// StatusService provides status and listing queries for deckhand environments.
type StatusService struct {
	lister ContainerLister
}

// NewStatusService creates a StatusService.
func NewStatusService(lister ContainerLister) *StatusService {
	return &StatusService{lister: lister}
}

// ProjectStatus returns all containers for the given project.
func (s *StatusService) ProjectStatus(projectName string) ([]domain.Container, error) {
	containers, err := s.lister.ListByProject(projectName)
	if err != nil {
		return nil, fmt.Errorf("querying project status: %w", err)
	}
	return containers, nil
}

// ListAll returns all deckhand-managed containers across all projects.
func (s *StatusService) ListAll() ([]domain.Container, error) {
	containers, err := s.lister.ListAll()
	if err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	return containers, nil
}
