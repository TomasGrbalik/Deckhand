package service_test

import (
	"errors"
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

type fakeConfigWriter struct {
	saved bool
	err   error
}

func (f *fakeConfigWriter) Save(_ string, _ *domain.Project) error {
	if f.err != nil {
		return f.err
	}
	f.saved = true
	return nil
}

type fakeRecreator struct {
	called bool
	err    error
}

func (f *fakeRecreator) Up(_ bool) error {
	if f.err != nil {
		return f.err
	}
	f.called = true
	return nil
}

func newTestPortService(proj *domain.Project) (*service.PortService, *fakeConfigWriter, *fakeRecreator) {
	cw := &fakeConfigWriter{}
	rec := &fakeRecreator{}
	svc := service.NewPortService(proj, "/tmp/test.yaml", cw, rec)
	return svc, cw, rec
}

func TestPortList(t *testing.T) {
	proj := &domain.Project{
		Ports: []domain.PortMapping{
			{Port: 8080, Name: "web", Protocol: "http"},
			{Port: 5432, Name: "pg", Protocol: "tcp", Internal: true},
		},
	}

	svc, _, _ := newTestPortService(proj)
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
	svc, _, _ := newTestPortService(proj)
	if len(svc.List()) != 0 {
		t.Error("expected empty list")
	}
}

func TestPortAdd(t *testing.T) {
	proj := &domain.Project{}
	svc, cw, rec := newTestPortService(proj)

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
	if !cw.saved {
		t.Error("expected config to be saved")
	}
	if !rec.called {
		t.Error("expected environment to be recreated")
	}
}

func TestPortAddDuplicate(t *testing.T) {
	proj := &domain.Project{
		Ports: []domain.PortMapping{{Port: 3000, Name: "api", Protocol: "http"}},
	}
	svc, _, _ := newTestPortService(proj)

	err := svc.Add(3000, "other", "http")
	if err == nil {
		t.Fatal("expected error for duplicate port")
	}
}

func TestPortAddInvalidPort(t *testing.T) {
	proj := &domain.Project{}
	svc, _, _ := newTestPortService(proj)

	for _, port := range []int{0, -1, 65536, 99999} {
		if err := svc.Add(port, "", "http"); err == nil {
			t.Errorf("expected error for port %d", port)
		}
	}
}

func TestPortAddInvalidProtocol(t *testing.T) {
	proj := &domain.Project{}
	svc, _, _ := newTestPortService(proj)

	err := svc.Add(3000, "", "banana")
	if err == nil {
		t.Fatal("expected error for invalid protocol")
	}
}

func TestPortAddSaveFails(t *testing.T) {
	proj := &domain.Project{}
	cw := &fakeConfigWriter{err: errors.New("disk full")}
	rec := &fakeRecreator{}
	svc := service.NewPortService(proj, "/tmp/test.yaml", cw, rec)

	err := svc.Add(3000, "api", "http")
	if err == nil {
		t.Fatal("expected error when save fails")
	}
	if rec.called {
		t.Error("environment should not be recreated when save fails")
	}
}

func TestPortAddUpFails(t *testing.T) {
	proj := &domain.Project{}
	cw := &fakeConfigWriter{}
	rec := &fakeRecreator{err: errors.New("compose failed")}
	svc := service.NewPortService(proj, "/tmp/test.yaml", cw, rec)

	err := svc.Add(3000, "api", "http")
	if err == nil {
		t.Fatal("expected error when recreate fails")
	}
	if !cw.saved {
		t.Error("config should have been saved before recreate attempted")
	}
}

func TestPortRemove(t *testing.T) {
	proj := &domain.Project{
		Ports: []domain.PortMapping{
			{Port: 8080, Name: "web", Protocol: "http"},
			{Port: 3000, Name: "api", Protocol: "http"},
		},
	}
	svc, cw, rec := newTestPortService(proj)

	if err := svc.Remove(8080); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	if len(proj.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(proj.Ports))
	}
	if proj.Ports[0].Port != 3000 {
		t.Errorf("remaining port = %d, want 3000", proj.Ports[0].Port)
	}
	if !cw.saved {
		t.Error("expected config to be saved")
	}
	if !rec.called {
		t.Error("expected environment to be recreated")
	}
}

func TestPortRemoveNotFound(t *testing.T) {
	proj := &domain.Project{}
	svc, _, _ := newTestPortService(proj)

	err := svc.Remove(9999)
	if err == nil {
		t.Fatal("expected error for nonexistent port")
	}
}
