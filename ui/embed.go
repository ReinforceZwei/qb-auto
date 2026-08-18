//go:build release

package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist returns the embedded frontend build, rooted at the dist folder.
//
// Only available when the binary is compiled with `-tags release` (the GitHub
// release workflow builds the frontend before compiling).
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
