package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestRequireSession(t *testing.T) {
	store := newSessionStore()
	store.put("valid-session", "user-1")
	auth := &authenticator{store: store, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("missing cookie denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/repo", nil)
		res := httptest.NewRecorder()
		auth.requireSession(next).ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid cookie passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/repo", nil)
		req.AddCookie(&http.Cookie{Name: "hayel_session", Value: "valid-session"})
		res := httptest.NewRecorder()
		auth.requireSession(next).ServeHTTP(res, req)
		if res.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
		}
	})
}

func TestLoginSetsStateCookie(t *testing.T) {
	auth := &authenticator{
		oauth: &oauth2.Config{
			ClientID:    "hayel-client",
			Endpoint:    oauth2.Endpoint{AuthURL: "https://issuer.example.com/auth"},
			RedirectURL: "https://git.example.com/auth/callback",
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		store:  newSessionStore(),
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	res := httptest.NewRecorder()
	auth.login(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	cookies := res.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a state cookie to be set")
	}
	if cookies[0].Name != "hayel_oauth_state" {
		t.Fatalf("cookie name = %q, want %q", cookies[0].Name, "hayel_oauth_state")
	}
	if cookies[0].Value == "" {
		t.Fatal("expected state cookie value to be non-empty")
	}
}
