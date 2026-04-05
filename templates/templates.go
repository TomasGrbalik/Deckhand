package templates

import "embed"

// FS contains all bundled template files, embedded at compile time.
// Each subdirectory is a template with metadata.yaml and Dockerfile.tmpl.
// The shared compose.yaml.tmpl lives at the root and is used by all
// templates unless a template provides its own override.
//
//go:embed compose.yaml.tmpl all:base all:python
var FS embed.FS
