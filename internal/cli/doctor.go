package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/config"
	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/infra/docker"
	"github.com/TomasGrbalik/deckhand/internal/infra/template"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate prerequisites and diagnose setup problems",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			svc := newDoctorService()
			results := svc.RunChecks(dir)

			w := cmd.OutOrStdout()
			for _, r := range results {
				fmt.Fprintf(w, "[%s] %s: %s\n", r.Status, r.Name, r.Message)
			}

			if service.HasFailures(results) {
				// Return a silent error — the check output already explains what failed.
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
				return errors.New("one or more checks failed")
			}
			return nil
		},
	}
}

// doctorDockerChecker adapts real Docker infra to the service.DockerChecker interface.
type doctorDockerChecker struct{}

// Ping implements service.DockerChecker.
func (d *doctorDockerChecker) Ping() error {
	client, err := docker.NewClient(context.Background())
	if err != nil {
		return err
	}
	_ = client.Close()
	return nil
}

// ComposeVersion implements service.DockerChecker.
func (d *doctorDockerChecker) ComposeVersion() (string, error) {
	return docker.NewCompose().ComposeVersion()
}

// doctorConfigLoader adapts real config loading to the service.ConfigLoader interface.
type doctorConfigLoader struct{}

// LoadGlobal implements service.ConfigLoader.
func (d *doctorConfigLoader) LoadGlobal() (*domain.GlobalConfig, error) {
	path, err := config.GlobalConfigPath()
	if err != nil {
		return nil, fmt.Errorf("resolving global config path: %w", err)
	}
	return config.LoadGlobal(path)
}

// LoadProject implements service.ConfigLoader.
func (d *doctorConfigLoader) LoadProject(dir string) (*domain.Project, error) {
	return config.Load(config.ProjectConfigPath(dir))
}

func newDoctorService() *service.DoctorService {
	return service.NewDoctorService(
		&doctorDockerChecker{},
		&doctorConfigLoader{},
		&template.EmbeddedSource{},
	)
}
