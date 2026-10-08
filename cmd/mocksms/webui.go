package main

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// uiHandler serves the embedded web UI bundle with SPA fallback: known
// files are served with a detected content type, every other path serves
// index.html so client-side routes resolve. A nil bundle (built without
// -tags embed) answers 404 with a hint to run task build.
func uiHandler(dist fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dist == nil {
			http.Error(w, "web UI not embedded in this build; run task build", http.StatusNotFound)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		data, err := fs.ReadFile(dist, name)
		if err != nil {
			data, err = fs.ReadFile(dist, "index.html")
			if err != nil {
				http.Error(w, "index.html missing from embedded bundle", http.StatusInternalServerError)
				return
			}
			name = "index.html"
		}
		if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write(data)
	})
}
