package main

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// uiContentTypes maps bundle extensions to content types explicitly. Go's
// mime package falls back to OS tables, which differ per platform; the UI
// must serve identical types everywhere.
var uiContentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".json":        "application/json",
	".map":         "application/json",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".ico":         "image/x-icon",
	".txt":         "text/plain; charset=utf-8",
	".webmanifest": "application/manifest+json",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
}

// uiContentType resolves the content type for a bundle file.
func uiContentType(name string) string {
	if contentType, ok := uiContentTypes[strings.ToLower(path.Ext(name))]; ok {
		return contentType
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

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
		if contentType := uiContentType(name); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write(data)
	})
}
