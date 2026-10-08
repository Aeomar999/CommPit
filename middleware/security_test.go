package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aeomar999/CommPit/config"
)

func TestSecurityMiddleware_TwilioExemptFromXMocksms(t *testing.T) {
	cfg := &config.SecurityConfig{
		AllowedHosts:    []string{},
		RequireXMocksms: true,
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := SecurityMiddleware(cfg)(next)

	t.Run("twilio adapter path passes without X-Mocksms header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/twilio/2010-04-01/Accounts/AC123/Messages.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("termii adapter path passes without X-Mocksms header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/termii/api/sms/send", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("native api path still requires X-Mocksms header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/sms", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
