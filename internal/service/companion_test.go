package service_test

import (
	"testing"

	"github.com/TomasGrbalik/deckhand/internal/service"
)

func TestCompanionRegistry_ListAvailable(t *testing.T) {
	reg := service.NewCompanionRegistry()
	services := reg.ListAvailable()

	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}

	// Results should be sorted by name.
	if services[0].Name != "postgres" {
		t.Errorf("services[0].Name: got %q, want %q", services[0].Name, "postgres")
	}
	if services[1].Name != "redis" {
		t.Errorf("services[1].Name: got %q, want %q", services[1].Name, "redis")
	}
}

func TestCompanionRegistry_ResolvePostgres(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc, err := reg.Resolve("postgres", "")
	if err != nil {
		t.Fatalf("Resolve(postgres) error: %v", err)
	}

	if svc.Image != "postgres:16-alpine" {
		t.Errorf("Image: got %q, want %q", svc.Image, "postgres:16-alpine")
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != 5432 {
		t.Errorf("Ports: got %v, want [5432]", svc.Ports)
	}
	if svc.HealthCheck.Test != "pg_isready -U dev" {
		t.Errorf("HealthCheck.Test: got %q, want %q", svc.HealthCheck.Test, "pg_isready -U dev")
	}
	if svc.Environment["POSTGRES_USER"] != "dev" {
		t.Errorf("Environment[POSTGRES_USER]: got %q, want %q", svc.Environment["POSTGRES_USER"], "dev")
	}
	if svc.Environment["POSTGRES_DB"] != "devdb" {
		t.Errorf("Environment[POSTGRES_DB]: got %q, want %q", svc.Environment["POSTGRES_DB"], "devdb")
	}
}

func TestCompanionRegistry_ResolveRedis(t *testing.T) {
	reg := service.NewCompanionRegistry()
	svc, err := reg.Resolve("redis", "")
	if err != nil {
		t.Fatalf("Resolve(redis) error: %v", err)
	}

	if svc.Image != "redis:7-alpine" {
		t.Errorf("Image: got %q, want %q", svc.Image, "redis:7-alpine")
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != 6379 {
		t.Errorf("Ports: got %v, want [6379]", svc.Ports)
	}
	if svc.HealthCheck.Test != "redis-cli ping" {
		t.Errorf("HealthCheck.Test: got %q, want %q", svc.HealthCheck.Test, "redis-cli ping")
	}
}

func TestCompanionRegistry_ResolveUnknown(t *testing.T) {
	reg := service.NewCompanionRegistry()
	_, err := reg.Resolve("mysql", "")
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}
}

func TestCompanionRegistry_ListAvailableFields(t *testing.T) {
	reg := service.NewCompanionRegistry()
	services := reg.ListAvailable()

	for _, svc := range services {
		if svc.Name == "" {
			t.Error("service Name should not be empty")
		}
		if svc.Description == "" {
			t.Errorf("service %q Description should not be empty", svc.Name)
		}
		if svc.Image == "" {
			t.Errorf("service %q Image should not be empty", svc.Name)
		}
		if len(svc.Ports) == 0 {
			t.Errorf("service %q should have at least one port", svc.Name)
		}
		if svc.HealthCheck.Test == "" {
			t.Errorf("service %q HealthCheck.Test should not be empty", svc.Name)
		}
	}
}
