package cli

import (
	"errors"
	"fmt"
	"os"

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
		newDoctorCmd(),
	)

	return cmd
}

// Execute runs the root command. Errors are passed through humanizeError
// to append actionable suggestions before display. If a subcommand returns
// errCanceled (user pressed Ctrl+C during an interactive prompt), it returns
// nil so main exits cleanly without printing an error.
func Execute() error {
	cmd := newRootCmd()
	// Silence Cobra's default error printing so we can humanize errors ourselves.
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err == nil {
		return nil
	}
	if errors.Is(err, errCanceled) {
		return nil
	}
	fmt.Fprintln(os.Stderr, "Error: "+humanizeError(err))
	return err
}
