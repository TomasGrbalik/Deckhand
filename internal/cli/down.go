package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the dev environment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc := newEnvironmentServiceForDown(*proj, dir)

			if err := svc.Down(); err != nil {
				return fmt.Errorf("stopping environment: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Environment stopped.")
			return nil
		},
	}
}
