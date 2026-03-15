package service_test

import (
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

func TestPortList(t *testing.T) {
	proj := &domain.Project{
		Ports: []domain.PortMapping{
			{Port: 8080, Name: "web", Protocol: "http"},
			{Port: 5432, Name: "pg", Protocol: "tcp", Internal: true},
		},
	}

	svc := service.NewPortService(proj)
	ports := svc.List()
	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(ports))
	}
	if ports[0].Port != 8080 {
		t.Errorf("first port = %d, want 8080", ports[0].Port)
	}
}

func TestPortListEmpty(t *testing.T) {
	proj := &domain.Project{}
	svc := service.NewPortService(proj)
	if len(svc.List()) != 0 {
		t.Error("expected empty list")
	}
}

func TestPortAdd(t *testing.T) {
	proj := &domain.Project{}
	svc := service.NewPortService(proj)

	if err := svc.Add(3000, "api", "http"); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	if len(proj.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(proj.Ports))
	}
	if proj.Ports[0].Port != 3000 {
		t.Errorf("Port = %d, want 3000", proj.Ports[0].Port)
	}
	if proj.Ports[0].Name != "api" {
		t.Errorf("Name = %q, want %q", proj.Ports[0].Name, "api")
	}
	if proj.Ports[0].Protocol != "http" {
		t.Errorf("Protocol = %q, want %q", proj.Ports[0].Protocol, "http")
	}
}

func TestPortAddDuplicate(t *testing.T) {
	proj := &domain.Project{
		Ports: []domain.PortMapping{{Port: 3000, Name: "api", Protocol: "http"}},
	}
	svc := service.NewPortService(proj)

	err := svc.Add(3000, "other", "http")
	if err == nil {
		t.Fatal("expected error for duplicate port")
	}
}

func TestPortRemove(t *testing.T) {
	proj := &domain.Project{
		Ports: []domain.PortMapping{
			{Port: 8080, Name: "web", Protocol: "http"},
			{Port: 3000, Name: "api", Protocol: "http"},
		},
	}
	svc := service.NewPortService(proj)

	if err := svc.Remove(8080); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	if len(proj.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(proj.Ports))
	}
	if proj.Ports[0].Port != 3000 {
		t.Errorf("remaining port = %d, want 3000", proj.Ports[0].Port)
	}
}

func TestPortAddInvalidPort(t *testing.T) {
	proj := &domain.Project{}
	svc := service.NewPortService(proj)

	for _, port := range []int{0, -1, 65536, 99999} {
		if err := svc.Add(port, "", "http"); err == nil {
			t.Errorf("expected error for port %d", port)
		}
	}
}

func TestPortAddInvalidProtocol(t *testing.T) {
	proj := &domain.Project{}
	svc := service.NewPortService(proj)

	err := svc.Add(3000, "", "banana")
	if err == nil {
		t.Fatal("expected error for invalid protocol")
	}
}

func TestPortRemoveNotFound(t *testing.T) {
	proj := &domain.Project{}
	svc := service.NewPortService(proj)

	err := svc.Remove(9999)
	if err == nil {
		t.Fatal("expected error for nonexistent port")
	}
}
