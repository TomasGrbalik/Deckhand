package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all deckhand environments on this host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, cleanup, err := newStatusService()
			if err != nil {
				return err
			}
			defer cleanup()

			containers, err := svc.ListAll()
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			if len(containers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No deckhand environments found.")
				return nil
			}

			// Group containers by project.
			projects := groupByProject(containers)

			out := cmd.OutOrStdout()
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tSTATUS\tSERVICES\tUPTIME")
			for _, p := range projects {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.name, p.status, p.services, p.uptime)
			}
			return w.Flush()
		},
	}
}

type projectSummary struct {
	name     string
	status   string
	services string
	uptime   string
}

func groupByProject(containers []domain.Container) []projectSummary {
	type projectData struct {
		services []string
		running  bool
		earliest time.Time
	}

	// Preserve insertion order with a slice of keys.
	var order []string
	projects := make(map[string]*projectData)

	for _, c := range containers {
		pd, exists := projects[c.Project]
		if !exists {
			pd = &projectData{}
			projects[c.Project] = pd
			order = append(order, c.Project)
		}
		pd.services = append(pd.services, c.Service)
		if c.State == "running" {
			pd.running = true
			if pd.earliest.IsZero() || c.Created.Before(pd.earliest) {
				pd.earliest = c.Created
			}
		}
	}

	result := make([]projectSummary, 0, len(order))
	for _, name := range order {
		pd := projects[name]
		status := "stopped"
		uptime := "—"
		if pd.running {
			status = "running"
			uptime = formatUptime(time.Since(pd.earliest))
		}
		result = append(result, projectSummary{
			name:     name,
			status:   status,
			services: strings.Join(pd.services, ","),
			uptime:   uptime,
		})
	}
	return result
}

func formatUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "<1m"
}
