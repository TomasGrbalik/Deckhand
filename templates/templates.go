package templates

import "embed"

// FS contains all bundled template files, embedded at compile time.
// Each subdirectory is a template with metadata.yaml, Dockerfile.tmpl,
// and compose.yaml.tmpl.
//
//go:embed all:base all:python
var FS embed.FS
