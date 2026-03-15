package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/config"
)

func newInitCmd() *cobra.Command {
	var templateName string
	var projectName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new .deckhand.yaml config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			cfgPath := config.ProjectConfigPath(dir)
			if _, err := os.Stat(cfgPath); err == nil {
				return fmt.Errorf(".deckhand.yaml already exists in %s", dir)
			}

			if projectName == "" {
				projectName = dirName(dir)
			}

			content := fmt.Sprintf("project: %s\ntemplate: %s\n", projectName, templateName)

			if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", cfgPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&templateName, "template", "base", "template to use")
	cmd.Flags().StringVar(&projectName, "project", "", "project name (default: directory name)")

	return cmd
}
