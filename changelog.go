// Package gobuildit holds files from the repository root that are embedded in the binary.
package gobuildit

import _ "embed"

// Changelog is CHANGELOG.md: go-build-it's own change log, shown in the app (FR-06a).
//
//go:embed CHANGELOG.md
var Changelog []byte
