package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestUIHandler(t *testing.T) {
	bundle := fstest.MapFS{
		"index.html":    {Data: []byte("<html>mock inbox</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
		"favicon.ico":   {Data: []byte("icon")},
	}

	t.Run("serves bundle files with content types", func(t *testing.T) {
		handler := uiHandler(bundle)
		cases := map[string]string{
			"/":              "text/html",
			"/index.html":    "text/html",
			"/assets/app.js": "text/javascript",
			"/favicon.ico":   "image/x-icon",
		}
		for path, contentType := range cases {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", path, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, contentType) {
				t.Errorf("%s: expected Content-Type %q, got %q", path, contentType, ct)
			}
		}
	})

	t.Run("unknown paths fall back to index.html", func(t *testing.T) {
		handler := uiHandler(bundle)
		req := httptest.NewRequest(http.MethodGet, "/inbox/threads", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "mock inbox") {
			t.Errorf("expected index.html fallback, got %q", rec.Body.String())
		}
	})

	t.Run("nil bundle explains task build", func(t *testing.T) {
		handler := uiHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "task build") {
			t.Errorf("expected build hint, got %q", rec.Body.String())
		}
	})
}
