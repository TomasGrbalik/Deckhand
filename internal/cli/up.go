package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	var build bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the dev environment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc, err := newEnvironmentService(*proj, dir)
			if err != nil {
				return err
			}

			if err := svc.Up(build); err != nil {
				return fmt.Errorf("starting environment: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Environment started.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&build, "build", false, "force image rebuild")

	return cmd
}
