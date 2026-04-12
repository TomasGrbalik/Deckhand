package service

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// CheckStatus represents the outcome of a single doctor check.
type CheckStatus string

const (
	// CheckPass indicates the check succeeded.
	CheckPass CheckStatus = "PASS"
	// CheckFail indicates the check failed.
	CheckFail CheckStatus = "FAIL"
	// CheckSkip indicates the check was not applicable.
	CheckSkip CheckStatus = "SKIP"
)

// CheckResult holds the outcome of a single doctor check.
type CheckResult struct {
	Name    string
	Status  CheckStatus
	Message string
}

// DockerChecker verifies Docker daemon reachability and Compose availability.
type DockerChecker interface {
	Ping() error
	ComposeVersion() (string, error)
	NetworkExists(name string) (bool, error)
}

// ConfigLoader abstracts config file loading so tests can inject fakes.
type ConfigLoader interface {
	LoadGlobal() (*domain.GlobalConfig, error)
	LoadProject(dir string) (*domain.Project, error)
}

// DoctorService validates that all deckhand prerequisites are met.
type DoctorService struct {
	docker   DockerChecker
	config   ConfigLoader
	template TemplateSource
}

// NewDoctorService creates a DoctorService with the given dependencies.
func NewDoctorService(docker DockerChecker, cfg ConfigLoader, tmpl TemplateSource) *DoctorService {
	return &DoctorService{
		docker:   docker,
		config:   cfg,
		template: tmpl,
	}
}

// RunChecks executes all prerequisite checks in order and returns the results.
// All checks run even if an earlier check fails (no short-circuiting).
func (s *DoctorService) RunChecks(projectDir string) []CheckResult {
	var results []CheckResult

	results = append(results, s.checkDocker())
	results = append(results, s.checkCompose())

	globalResult, globalCfg := s.checkGlobalConfigWithResult()
	results = append(results, globalResult)

	projResult, proj := s.checkProjectConfig(projectDir)
	results = append(results, projResult)

	results = append(results, s.checkTemplate(proj))
	results = append(results, s.checkNetwork(globalCfg))

	return results
}

// HasFailures returns true if any check result has a FAIL status.
func HasFailures(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == CheckFail {
			return true
		}
	}
	return false
}

func (s *DoctorService) checkDocker() CheckResult {
	if err := s.docker.Ping(); err != nil {
		return CheckResult{
			Name:    "Docker daemon",
			Status:  CheckFail,
			Message: err.Error(),
		}
	}
	return CheckResult{
		Name:    "Docker daemon",
		Status:  CheckPass,
		Message: "daemon is reachable",
	}
}

func (s *DoctorService) checkCompose() CheckResult {
	version, err := s.docker.ComposeVersion()
	if err != nil {
		return CheckResult{
			Name:    "Compose V2",
			Status:  CheckFail,
			Message: err.Error(),
		}
	}
	return CheckResult{
		Name:    "Compose V2",
		Status:  CheckPass,
		Message: "version " + version,
	}
}

// checkGlobalConfigWithResult returns the check result and the loaded config.
// The config is passed to checkNetwork.
func (s *DoctorService) checkGlobalConfigWithResult() (CheckResult, *domain.GlobalConfig) {
	cfg, err := s.config.LoadGlobal()
	if err != nil {
		return CheckResult{
			Name:    "Global config",
			Status:  CheckFail,
			Message: err.Error(),
		}, nil
	}
	return CheckResult{
		Name:    "Global config",
		Status:  CheckPass,
		Message: "valid",
	}, cfg
}

// checkProjectConfig returns the check result and the loaded project (nil if
// not found or invalid). The project is passed to checkTemplate.
func (s *DoctorService) checkProjectConfig(dir string) (CheckResult, *domain.Project) {
	proj, err := s.config.LoadProject(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return CheckResult{
				Name:    "Project config",
				Status:  CheckSkip,
				Message: "no .deckhand.yaml in current directory",
			}, nil
		}
		return CheckResult{
			Name:    "Project config",
			Status:  CheckFail,
			Message: err.Error(),
		}, nil
	}
	return CheckResult{
		Name:    "Project config",
		Status:  CheckPass,
		Message: "valid (" + proj.Name + ")",
	}, proj
}

func (s *DoctorService) checkTemplate(proj *domain.Project) CheckResult {
	if proj == nil {
		return CheckResult{
			Name:    "Template",
			Status:  CheckSkip,
			Message: "no project config loaded",
		}
	}
	_, _, err := s.template.Load(proj.Template)
	if err != nil {
		return CheckResult{
			Name:    "Template",
			Status:  CheckFail,
			Message: "template " + proj.Template + " not found",
		}
	}
	return CheckResult{
		Name:    "Template",
		Status:  CheckPass,
		Message: "template " + proj.Template + " found",
	}
}

func (s *DoctorService) checkNetwork(cfg *domain.GlobalConfig) CheckResult {
	if cfg == nil || !cfg.Network.IsConfigured() {
		return CheckResult{
			Name:    "Docker network",
			Status:  CheckSkip,
			Message: "no network configured in global config",
		}
	}

	net := cfg.Network
	exists, err := s.docker.NetworkExists(net.Name)
	if err != nil {
		return CheckResult{
			Name:    "Docker network",
			Status:  CheckFail,
			Message: err.Error(),
		}
	}
	if !exists {
		return CheckResult{
			Name:   "Docker network",
			Status: CheckFail,
			Message: fmt.Sprintf("network %q not found — create it with:\n  docker network create --driver=bridge --subnet=%s --gateway=%s %s",
				net.Name, net.Subnet, net.Gateway, net.Name),
		}
	}

	return CheckResult{
		Name:    "Docker network",
		Status:  CheckPass,
		Message: "network " + net.Name + " exists",
	}
}
