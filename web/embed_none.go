//go:build !embed

package web

import "io/fs"

// Dist is nil without the embed tag; cmd/mocksms serves a 404 explaining
// to run task build. The UI is only embedded in release builds.
var Dist fs.FS
