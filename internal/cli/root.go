package cli

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var verbose bool

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deckhand",
		Short:   "Orchestrate Docker-based dev environments on remote servers",
		Version: Version,
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	cmd.AddCommand(
		newInitCmd(),
		newUpCmd(),
		newDownCmd(),
		newDestroyCmd(),
		newShellCmd(),
		newExecCmd(),
		newLogsCmd(),
	)

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
