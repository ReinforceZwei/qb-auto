//go:build !release

package ui

import (
	"embed"
	"io/fs"
)

//go:embed stub/index.html
var stubFS embed.FS

// Dist returns a minimal placeholder file system (a single index.html) so the
// app compiles without a built frontend. Local development serves the real UI
// from the Vite dev server instead.
func Dist() fs.FS {
	return stubFS
}
