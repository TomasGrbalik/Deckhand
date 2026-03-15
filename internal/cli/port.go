package cli

import (
	"fmt"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/config"
	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

func newPortCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "port",
		Short: "Manage port mappings",
	}

	cmd.AddCommand(
		newPortListCmd(),
		newPortAddCmd(),
		newPortRemoveCmd(),
	)

	return cmd
}

func newPortListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all port mappings for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			// List is read-only — no config writer or env recreator needed.
			svc := service.NewPortService(proj, "", nil, nil)
			ports := svc.List()

			out := cmd.OutOrStdout()

			if len(ports) == 0 {
				fmt.Fprintln(out, "No port mappings configured.")
				return nil
			}

			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "PORT\tNAME\tPROTOCOL\tACCESS")
			for _, p := range ports {
				access := fmt.Sprintf("ssh -L %d:localhost:%d", p.Port, p.Port)
				if p.Internal {
					access = "internal only"
				}
				name := p.Name
				if name == "" {
					name = "—"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.Port, name, p.Protocol, access)
			}
			return w.Flush()
		},
	}
}

func newPortAddCmd() *cobra.Command {
	var name string
	var protocol string

	cmd := &cobra.Command{
		Use:   "add <port>",
		Short: "Add a port mapping",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid port number: %w", err)
			}

			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc := newPortService(proj, dir)
			if err := svc.Add(port, name, protocol); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added port %d.\n", port)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "human-readable label for the port")
	cmd.Flags().StringVar(&protocol, "protocol", "http", "protocol (http or tcp)")

	return cmd
}

func newPortRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <port>",
		Short: "Remove a port mapping",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid port number: %w", err)
			}

			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc := newPortService(proj, dir)
			if err := svc.Remove(port); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed port %d.\n", port)
			return nil
		},
	}
}

// newPortService creates a PortService wired to real config persistence
// and environment recreation.
func newPortService(proj *domain.Project, dir string) *service.PortService {
	cfgPath := config.ProjectConfigPath(dir)
	envSvc := newEnvironmentService(*proj, dir)
	return service.NewPortService(proj, cfgPath, configSaver{}, envSvc)
}

// configSaver implements service.ConfigWriter using config.Save.
type configSaver struct{}

// Save implements service.ConfigWriter.
func (configSaver) Save(path string, proj *domain.Project) error {
	return config.Save(path, proj)
}
