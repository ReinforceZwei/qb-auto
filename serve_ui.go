package main

import (
	"io/fs"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// registerFrontend serves the embedded SPA on all non-API routes.
//
// `apis.Static` with indexFallback=true renders index.html for any missing
// path, so client-side routes (e.g. /history) keep working on refresh.
// PocketBase's own /api/* and /_/* routes are more specific and therefore
// take precedence, so they are never shadowed.
func registerFrontend(se *core.ServeEvent, fsys fs.FS) {
	se.Router.GET("/{path...}", apis.Static(fsys, true))
}
