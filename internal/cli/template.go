package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/infra/template"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage templates",
	}

	cmd.AddCommand(newTemplateListCmd())

	return cmd
}

func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			embedded := &template.EmbeddedSource{}

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolving home directory: %w", err)
			}
			userDir := filepath.Join(home, ".config", "deckhand", "templates")
			fs := &template.FilesystemSource{Dir: userDir}

			registry := service.NewTemplateRegistry(embedded, fs)

			templates, err := registry.List()
			if err != nil {
				return fmt.Errorf("listing templates: %w", err)
			}

			if len(templates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No templates found.")
				return nil
			}

			out := cmd.OutOrStdout()
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tSOURCE")
			for _, t := range templates {
				fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Description, t.Source)
			}
			return w.Flush()
		},
	}
}
