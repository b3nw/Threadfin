// Package webui embeds the compiled single-page web interface.
// The sources live in web/; `npm run build` (Vite) writes the production
// bundle to webui/dist, which is served by the Threadfin binary at /ui/.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built web interface with the dist/ prefix stripped, so
// index.html sits at the root.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
