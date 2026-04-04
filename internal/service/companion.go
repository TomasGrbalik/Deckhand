package service

import (
	"fmt"
	"sort"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// CompanionRegistry provides access to the hardcoded set of companion services.
type CompanionRegistry struct {
	services map[string]domain.CompanionService
}

// NewCompanionRegistry creates a registry pre-loaded with the built-in
// companion service definitions (postgres, redis).
func NewCompanionRegistry() *CompanionRegistry {
	return &CompanionRegistry{
		services: map[string]domain.CompanionService{
			"postgres": {
				Name:        "postgres",
				Description: "PostgreSQL relational database",
				Image:       "postgres:16-alpine",
				Ports:       []int{5432},
				Environment: map[string]string{
					"POSTGRES_USER":     "dev",
					"POSTGRES_PASSWORD": "dev",
					"POSTGRES_DB":       "devdb",
				},
				HealthCheck: domain.HealthCheck{
					Test:     "pg_isready -U dev",
					Interval: "5s",
					Timeout:  "3s",
					Retries:  5,
				},
				Volumes: []string{"postgres-data:/var/lib/postgresql/data"},
			},
			"redis": {
				Name:        "redis",
				Description: "Redis in-memory data store",
				Image:       "redis:7-alpine",
				Ports:       []int{6379},
				Environment: map[string]string{},
				HealthCheck: domain.HealthCheck{
					Test:     "redis-cli ping",
					Interval: "5s",
					Timeout:  "3s",
					Retries:  5,
				},
				Volumes: []string{"redis-data:/data"},
			},
		},
	}
}

// ListAvailable returns all registered companion services, sorted by name.
func (r *CompanionRegistry) ListAvailable() []domain.CompanionService {
	result := make([]domain.CompanionService, 0, len(r.services))
	for _, svc := range r.services {
		result = append(result, svc)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Resolve looks up a companion service by name. The version parameter is
// accepted for future use but currently ignored.
func (r *CompanionRegistry) Resolve(name, _version string) (domain.CompanionService, error) {
	svc, ok := r.services[name]
	if !ok {
		return domain.CompanionService{}, fmt.Errorf("unknown companion service: %q", name)
	}
	return svc, nil
}
