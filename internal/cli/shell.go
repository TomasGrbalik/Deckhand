package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	var serviceName string
	var shellCmd string

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Open an interactive shell in a container",
		RunE: func(_ *cobra.Command, _ []string) error {
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

			// exec_user is a devcontainer-scoped property of the project's
			// template. Companion services (postgres, redis, ...) have their
			// own image users and shouldn't inherit it.
			var execUser string
			if serviceName == "devcontainer" {
				execUser, err = resolveExecUser(dir, proj.Template)
				if err != nil {
					return err
				}
			}

			if err := svc.Shell(proj.Name, serviceName, strings.Fields(shellCmd), execUser); err != nil {
				return fmt.Errorf("shell: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&serviceName, "service", "devcontainer", "target service name")
	cmd.Flags().StringVar(&shellCmd, "cmd", "zsh", "shell command to run")

	return cmd
}
