package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var follow bool
	var tail int

	cmd := &cobra.Command{
		Use:   "logs [service]",
		Short: "Stream container logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			serviceName := "devcontainer"
			if len(args) > 0 {
				serviceName = args[0]
			}

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

			rc, err := svc.Logs(proj.Name, serviceName, follow, strconv.Itoa(tail))
			if err != nil {
				return fmt.Errorf("logs: %w", err)
			}
			defer rc.Close()

			if _, err := io.Copy(os.Stdout, rc); err != nil {
				return fmt.Errorf("streaming logs: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	cmd.Flags().IntVar(&tail, "tail", 100, "number of lines to show from the end")

	return cmd
}
