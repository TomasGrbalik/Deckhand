package cli

import (
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of the current project's containers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc, cleanup, err := newStatusService()
			if err != nil {
				return err
			}
			defer cleanup()

			containers, err := svc.ProjectStatus(proj.Name)
			if err != nil {
				return fmt.Errorf("status: %w", err)
			}

			out := cmd.OutOrStdout()

			if len(containers) == 0 {
				fmt.Fprintf(out, "No containers found for project %q.\n", proj.Name)
				return nil
			}

			// Determine overall project state from container states.
			projectState := "stopped"
			for _, c := range containers {
				if c.State == "running" {
					projectState = "running"
					break
				}
			}

			fmt.Fprintf(out, "PROJECT: %s (%s)\n\n", proj.Name, projectState)

			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "SERVICE\tIMAGE\tSTATUS\tHEALTH\tPORTS")
			for _, c := range containers {
				health := c.Health
				if health == "" {
					health = "—"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", c.Service, c.Image, c.Status, health, formatPorts(c.Ports))
			}
			return w.Flush()
		},
	}
}

// formatPorts renders port mappings. Shows just the public port when
// public == private, and public:private when they differ.
func formatPorts(ports []domain.ContainerPort) string {
	if len(ports) == 0 {
		return "—"
	}
	strs := make([]string, len(ports))
	for i, p := range ports {
		if p.Public == p.Private {
			strs[i] = strconv.Itoa(p.Public)
		} else {
			strs[i] = strconv.Itoa(p.Public) + ":" + strconv.Itoa(p.Private)
		}
	}
	return strings.Join(strs, ", ")
}
