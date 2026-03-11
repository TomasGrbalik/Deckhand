package templates

import "embed"

// FS contains all bundled template files, embedded at compile time.
//
//go:embed all:base
var FS embed.FS
