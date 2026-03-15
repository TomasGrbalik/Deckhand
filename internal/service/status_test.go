package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

type fakeLister struct {
	byProject map[string][]domain.Container
	all       []domain.Container
	err       error
}

func (f *fakeLister) ListByProject(projectName string) ([]domain.Container, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byProject[projectName], nil
}

func (f *fakeLister) ListAll() ([]domain.Container, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.all, nil
}

func TestProjectStatus(t *testing.T) {
	containers := []domain.Container{
		{
			ID:      "abc123",
			Name:    "myproject-devcontainer-1",
			Service: "devcontainer",
			Project: "myproject",
			Image:   "myproject:latest",
			State:   "running",
			Status:  "Up 2 hours",
			Created: time.Now().Add(-2 * time.Hour),
			Ports:   []domain.ContainerPort{{Public: 8080, Private: 8080}},
		},
	}

	lister := &fakeLister{
		byProject: map[string][]domain.Container{
			"myproject": containers,
		},
	}

	svc := service.NewStatusService(lister)
	got, err := svc.ProjectStatus("myproject")
	if err != nil {
		t.Fatalf("ProjectStatus() error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 container, got %d", len(got))
	}
	if got[0].Service != "devcontainer" {
		t.Errorf("Service = %q, want %q", got[0].Service, "devcontainer")
	}
	if got[0].State != "running" {
		t.Errorf("State = %q, want %q", got[0].State, "running")
	}
}

func TestProjectStatusEmpty(t *testing.T) {
	lister := &fakeLister{
		byProject: map[string][]domain.Container{},
	}

	svc := service.NewStatusService(lister)
	got, err := svc.ProjectStatus("nonexistent")
	if err != nil {
		t.Fatalf("ProjectStatus() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 containers, got %d", len(got))
	}
}

func TestListAll(t *testing.T) {
	containers := []domain.Container{
		{Project: "proj-a", Service: "devcontainer", State: "running"},
		{Project: "proj-b", Service: "devcontainer", State: "exited"},
	}

	lister := &fakeLister{all: containers}
	svc := service.NewStatusService(lister)

	got, err := svc.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(got))
	}
}

func TestStatusServiceError(t *testing.T) {
	lister := &fakeLister{err: errors.New("docker unavailable")}
	svc := service.NewStatusService(lister)

	_, err := svc.ProjectStatus("test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	_, err = svc.ListAll()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
