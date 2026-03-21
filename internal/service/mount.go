package service

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/TomasGrbalik/deckhand/internal/domain"
)

// MergeMounts combines mounts from three sources (template defaults, global
// config, project config) into a single resolved set. Later sources win when
// mounts share the same name. Env var references (${VAR}) and tilde paths (~)
// in source fields are resolved against the current host environment. Mounts
// whose sources cannot be resolved are skipped and reported as warnings.
func MergeMounts(template, global, project domain.Mounts) (domain.Mounts, []string) {
	volumes := mergeByName(template.Volumes, global.Volumes, project.Volumes)
	secrets := mergeByName(template.Secrets, global.Secrets, project.Secrets)
	sockets := mergeByName(template.Sockets, global.Sockets, project.Sockets)

	var warnings []string

	resolvedSecrets := resolveSecrets(secrets, &warnings)
	resolvedSockets := resolveSockets(sockets, &warnings)

	return domain.Mounts{
		Volumes: volumes,
		Secrets: resolvedSecrets,
		Sockets: resolvedSockets,
	}, warnings
}

// named is a constraint for mount types that have a Name field and an Enabled
// pointer.
type named interface {
	domain.VolumeMount | domain.SecretMount | domain.SocketMount
}

func getName[T named](m T) string {
	switch v := any(m).(type) {
	case domain.VolumeMount:
		return v.Name
	case domain.SecretMount:
		return v.Name
	case domain.SocketMount:
		return v.Name
	}
	return ""
}

func getEnabled[T named](m T) *bool {
	switch v := any(m).(type) {
	case domain.VolumeMount:
		return v.Enabled
	case domain.SecretMount:
		return v.Enabled
	case domain.SocketMount:
		return v.Enabled
	}
	return nil
}

// mergeByName deduplicates mounts across three layers by name, where later
// layers replace earlier ones. Mounts with enabled=false are removed from the
// final result.
func mergeByName[T named](layers ...[]T) []T {
	type entry struct {
		mount T
		order int
	}
	seen := make(map[string]entry)
	idx := 0

	for _, layer := range layers {
		for _, m := range layer {
			name := getName(m)
			if _, exists := seen[name]; !exists {
				seen[name] = entry{mount: m, order: idx}
				idx++
			} else {
				seen[name] = entry{mount: m, order: seen[name].order}
			}
		}
	}

	// Collect results preserving insertion order, filtering out disabled mounts.
	result := make([]T, 0, len(seen))
	sorted := make([]entry, 0, len(seen))
	for _, e := range seen {
		sorted = append(sorted, e)
	}
	// Sort by insertion order.
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].order < sorted[i].order {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for _, e := range sorted {
		enabled := getEnabled(e.mount)
		if enabled != nil && !*enabled {
			continue
		}
		result = append(result, e.mount)
	}

	return result
}

// resolveSource expands env var references and tilde paths in a source string.
// Returns the resolved value, or an error describing why it couldn't resolve.
func resolveSource(source string) (string, error) {
	if source == "" {
		return "", nil
	}

	if strings.HasPrefix(source, "${") {
		return resolveEnvVar(source)
	}

	return resolveFilePath(source)
}

// resolveEnvVar expands a ${VAR} reference from the host environment.
func resolveEnvVar(source string) (string, error) {
	// Extract variable name from ${VAR} pattern.
	if !strings.HasSuffix(source, "}") {
		return "", fmt.Errorf("malformed env var reference: %s", source)
	}
	varName := source[2 : len(source)-1]
	value := os.Getenv(varName)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", varName)
	}
	return value, nil
}

// resolveFilePath expands tilde and checks that the file exists.
func resolveFilePath(source string) (string, error) {
	expanded := expandTilde(source)

	if _, err := os.Stat(expanded); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("source file does not exist: %s", expanded)
		}
		return "", fmt.Errorf("cannot access source file %s: %w", expanded, err)
	}

	return expanded, nil
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}

// resolveSecrets resolves source fields on secrets and filters out
// unresolvable ones, appending warnings for each skip.
func resolveSecrets(secrets []domain.SecretMount, warnings *[]string) []domain.SecretMount {
	var result []domain.SecretMount
	for _, s := range secrets {
		if s.Source == "" {
			result = append(result, s)
			continue
		}
		resolved, err := resolveSource(s.Source)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("skipping secret %q: %s", s.Name, err))
			continue
		}
		s.Source = resolved
		result = append(result, s)
	}
	return result
}

// resolveSockets resolves source fields on sockets and filters out
// unresolvable ones, appending warnings for each skip.
func resolveSockets(sockets []domain.SocketMount, warnings *[]string) []domain.SocketMount {
	var result []domain.SocketMount
	for _, s := range sockets {
		if s.Source == "" {
			result = append(result, s)
			continue
		}
		resolved, err := resolveSource(s.Source)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("skipping socket %q: %s", s.Name, err))
			continue
		}
		s.Source = resolved
		result = append(result, s)
	}
	return result
}
