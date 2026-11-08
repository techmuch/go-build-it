// Package web embeds the built React frontend (FR-01).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend files.
func Dist() fs.FS { sub, _ := fs.Sub(dist, "dist"); return sub }
