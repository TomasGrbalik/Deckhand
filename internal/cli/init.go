package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/TomasGrbalik/deckhand/internal/config"
	"github.com/TomasGrbalik/deckhand/internal/domain"
	"github.com/TomasGrbalik/deckhand/internal/infra/template"
	"github.com/TomasGrbalik/deckhand/internal/service"
)

// errCanceled is returned when the user aborts an interactive prompt (Ctrl+C).
// The root command checks for this sentinel to exit cleanly without printing
// an error message.
var errCanceled = errors.New("canceled")

func newInitCmd() *cobra.Command {
	var templateFlag string
	var projectFlag string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new .deckhand.yaml config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := projectDir()
			if err != nil {
				return err
			}

			cfgPath := config.ProjectConfigPath(dir)
			if _, statErr := os.Stat(cfgPath); statErr == nil {
				return fmt.Errorf(".deckhand.yaml already exists in %s", dir)
			}

			initSvc := newInitService()

			// --- Template selection ---
			templateName := templateFlag
			templateFlagSet := cmd.Flags().Changed("template")

			if !templateFlagSet {
				templateName, err = pickTemplate(initSvc)
				if err != nil {
					return err
				}
			}

			// Validate template and load metadata.
			meta, err := initSvc.ResolveTemplate(templateName)
			if err != nil {
				return err
			}

			// --- Variable editing ---
			variables := initSvc.DefaultVariables(meta)

			// Only prompt for variables if there are any and template wasn't
			// provided via flag (flags mean "use defaults").
			if len(meta.Variables) > 0 && !templateFlagSet {
				variables, err = editVariables(initSvc, meta, variables)
				if err != nil {
					return err
				}
			}

			// --- Companion service selection ---
			var selectedServices []string
			if !templateFlagSet {
				selectedServices, err = pickCompanions(initSvc)
				if err != nil {
					return err
				}
			}

			// --- Project name ---
			projectName := projectFlag
			if !cmd.Flags().Changed("project") {
				defaultName := dirName(dir)
				projectName, err = promptProjectName(defaultName)
				if err != nil {
					return err
				}
			}

			// Build and save.
			proj := initSvc.BuildProject(projectName, templateName, variables, meta, selectedServices)
			if err := config.Save(cfgPath, proj); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", cfgPath)

			// Display global mount summary (informational only).
			displayGlobalMountSummary(cmd.OutOrStdout())

			return nil
		},
	}

	cmd.Flags().StringVar(&templateFlag, "template", "", "template to use (skips interactive picker)")
	cmd.Flags().StringVar(&projectFlag, "project", "", "project name (default: directory name)")

	return cmd
}

// newInitService creates an InitService wired to real template sources.
// The composite source tries user templates (filesystem) before falling back
// to embedded, so user templates can override builtins for metadata loading.
func newInitService() *service.InitService {
	embedded := &template.EmbeddedSource{}
	companions := service.NewCompanionRegistry()

	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to embedded-only if we can't resolve home.
		return service.NewInitService(
			service.NewTemplateRegistry(embedded),
			embedded,
			companions,
		)
	}
	userDir := filepath.Join(home, ".config", "deckhand", "templates")
	fs := &template.FilesystemSource{Dir: userDir}
	registry := service.NewTemplateRegistry(embedded, fs)
	source := &compositeSource{sources: []service.TemplateSource{fs, embedded}}

	return service.NewInitService(registry, source, companions)
}

// pickTemplate shows an interactive template picker. If only one template
// is available, it auto-selects it.
func pickTemplate(svc *service.InitService) (string, error) {
	templates, err := svc.ListTemplates()
	if err != nil {
		return "", fmt.Errorf("listing templates: %w", err)
	}

	if len(templates) == 0 {
		return "", errors.New("no templates found (run `deckhand template list` to check)")
	}

	// Auto-select if only one template.
	if len(templates) == 1 {
		return templates[0].Name, nil
	}

	options := make([]huh.Option[string], len(templates))
	for i, t := range templates {
		label := t.Name
		if t.Description != "" {
			label = fmt.Sprintf("%s — %s", t.Name, t.Description)
		}
		options[i] = huh.NewOption(label, t.Name)
	}

	var selected string
	err = huh.NewSelect[string]().
		Title("Select a template").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errCanceled
		}
		return "", fmt.Errorf("template selection: %w", err)
	}

	return selected, nil
}

// editVariables shows input prompts for each template variable, pre-filled
// with defaults.
func editVariables(svc *service.InitService, meta *domain.TemplateMeta, defaults map[string]string) (map[string]string, error) {
	names := svc.SortedVariableNames(meta)

	// Go doesn't allow taking the address of a map value, so we use a
	// parallel slice of string pointers that huh can write into.
	values := make([]*string, len(names))
	for i, name := range names {
		v := defaults[name]
		values[i] = &v
	}

	fields := make([]huh.Field, 0, len(names))
	for i, name := range names {
		varDef := meta.Variables[name]
		title := name
		if varDef.Description != "" {
			title = fmt.Sprintf("%s (%s)", name, varDef.Description)
		}

		fields = append(fields, huh.NewInput().
			Title(title).
			Value(values[i]).
			Placeholder(varDef.Default))
	}

	err := huh.NewForm(huh.NewGroup(fields...)).Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, errCanceled
		}
		return nil, fmt.Errorf("variable input: %w", err)
	}

	// Collect results back into a map.
	result := make(map[string]string, len(names))
	for i, name := range names {
		result[name] = *values[i]
	}

	return result, nil
}

// pickCompanions shows an interactive multi-select for companion services.
// Returns the selected service names, or nil if none are available.
func pickCompanions(svc *service.InitService) ([]string, error) {
	companions := svc.ListCompanions()
	if len(companions) == 0 {
		return nil, nil
	}

	options := make([]huh.Option[string], len(companions))
	for i, c := range companions {
		label := fmt.Sprintf("%s — %s", c.Name, c.Description)
		options[i] = huh.NewOption(label, c.Name)
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("Select companion services (optional)").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, errCanceled
		}
		return nil, fmt.Errorf("companion selection: %w", err)
	}

	return selected, nil
}

// promptProjectName asks for the project name with a default.
func promptProjectName(defaultName string) (string, error) {
	name := defaultName
	err := huh.NewInput().
		Title("Project name").
		Value(&name).
		Placeholder(defaultName).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errCanceled
		}
		return "", fmt.Errorf("project name input: %w", err)
	}

	return name, nil
}
