package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func newDestroyCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the dev environment and remove all generated files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				var confirmed bool
				err := huh.NewConfirm().
					Title("Destroy the dev environment?").
					Description("This will stop all containers, remove volumes, and delete .deckhand/.").
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

			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc := newEnvironmentService(*proj, dir)

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
