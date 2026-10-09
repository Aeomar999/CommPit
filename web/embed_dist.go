//go:build embed

package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFiles embed.FS

// Dist is the built web UI bundle, served at / by cmd/mocksms when built
// with -tags embed (see task build).
var Dist = mustSubFS(distFiles)

func mustSubFS(files embed.FS) fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic("web: embedded dist bundle is missing")
	}
	return sub
}
