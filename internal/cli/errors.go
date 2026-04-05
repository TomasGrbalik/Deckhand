package cli

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/TomasGrbalik/deckhand/internal/service"
)

// humanizeError inspects err and returns a user-friendly message with an
// actionable suggestion appended. If the error does not match a known
// pattern, the original error string is returned unchanged.
func humanizeError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	// Docker daemon not running / not accessible.
	if containsAny(msg, "connection refused", "Cannot connect to the Docker daemon",
		"pinging docker daemon", "creating docker client") ||
		(strings.Contains(msg, "permission denied") && strings.Contains(msg, "docker")) {
		return msg + "\n\nIs Docker running? Start the Docker daemon and try again."
	}

	// Project config does not exist — suggest deckhand init.
	if strings.Contains(msg, "loading config") && errors.Is(err, fs.ErrNotExist) {
		return msg + "\n\nRun `deckhand init` to create a project config."
	}

	// No running environment (no compose file from a prior `up`).
	if errors.Is(err, service.ErrNoEnvironment) {
		return msg + "\n\nRun `deckhand up` first."
	}

	// Template not found.
	if strings.Contains(msg, "template") && errors.Is(err, fs.ErrNotExist) {
		return msg + "\n\nCheck available templates with `deckhand template list`."
	}

	return msg
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
