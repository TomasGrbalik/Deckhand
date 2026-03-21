package cli

import (
	"errors"

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
	cmd.CompletionOptions.DisableDefaultCmd = true

	cmd.AddCommand(
		newInitCmd(),
		newUpCmd(),
		newDownCmd(),
		newDestroyCmd(),
		newShellCmd(),
		newExecCmd(),
		newLogsCmd(),
		newStatusCmd(),
		newListCmd(),
		newPortCmd(),
		newConnectCmd(),
		newTemplateCmd(),
	)

	return cmd
}

// Execute runs the root command. If a subcommand returns errCanceled
// (user pressed Ctrl+C during an interactive prompt), it returns nil
// so main exits cleanly without printing an error.
func Execute() error {
	err := newRootCmd().Execute()
	if errors.Is(err, errCanceled) {
		return nil
	}
	return err
}
