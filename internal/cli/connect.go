package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newConnectCmd() *cobra.Command {
	var host string

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Print the SSH tunnel command for this project's ports",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if host == "" {
				return errors.New("--host is required (e.g. --host user@myserver)")
			}

			dir, err := projectDir()
			if err != nil {
				return err
			}

			proj, err := loadProject(dir)
			if err != nil {
				return err
			}

			// Collect non-internal ports.
			var tunnels []string
			for _, p := range proj.Ports {
				if !p.Internal {
					tunnels = append(tunnels, "-L "+strconv.Itoa(p.Port)+":localhost:"+strconv.Itoa(p.Port))
				}
			}

			out := cmd.OutOrStdout()

			if len(tunnels) == 0 {
				fmt.Fprintln(out, "No external ports to tunnel.")
				return nil
			}

			fmt.Fprintf(out, "ssh -N %s %s\n", strings.Join(tunnels, " "), host)
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "SSH target (e.g. user@myserver)")

	return cmd
}
