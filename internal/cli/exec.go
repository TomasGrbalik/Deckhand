package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec [command] [args...]",
		Short: "Run a command in the devcontainer",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			svc, cleanup, err := newContainerService()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := svc.Exec(proj.Name, "devcontainer", args); err != nil {
				return fmt.Errorf("exec: %w", err)
			}

			return nil
		},
	}
}
