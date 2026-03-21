package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// formatVolumeEntry formats a volume name with its size for display.
func formatVolumeEntry(name string, size int64) string {
	if size < 0 {
		return name
	}
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case size >= gb:
		return fmt.Sprintf("%s (%.1f GB)", name, float64(size)/float64(gb))
	case size >= mb:
		return fmt.Sprintf("%s (%.1f MB)", name, float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf("%s (%.1f KB)", name, float64(size)/float64(kb))
	default:
		return fmt.Sprintf("%s (%d B)", name, size)
	}
}

func newDestroyCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the dev environment and remove all generated files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc, cleanup, err := newEnvironmentServiceWithVolumes(*proj, dir)
			if err != nil {
				return err
			}
			defer cleanup()

			if !yes {
				// Discover volumes to show in the prompt.
				vols, err := svc.ProjectVolumes()
				if err != nil {
					return fmt.Errorf("listing project volumes: %w", err)
				}

				description := "This will stop all containers, remove volumes, and delete .deckhand/."
				if len(vols) > 0 {
					entries := make([]string, len(vols))
					for i, v := range vols {
						entries[i] = formatVolumeEntry(v.Name, v.Size)
					}
					description = "This will stop all containers, delete .deckhand/, and remove volumes:\n  " +
						strings.Join(entries, "\n  ")
				}

				var confirmed bool
				err = huh.NewConfirm().
					Title("Destroy the dev environment?").
					Description(description).
					Affirmative("Yes, destroy").
					Negative("Cancel").
					Value(&confirmed).
					Run()
				if err != nil {
					return fmt.Errorf("confirmation prompt: %w", err)
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}

			if err := svc.Destroy(); err != nil {
				return fmt.Errorf("destroying environment: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Environment destroyed.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")

	return cmd
}
