package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type authenticator struct {
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
	store    *sessionStore
	logger   *slog.Logger
}

func newAuthenticator(oauthConfig *oauth2.Config, verifier *oidc.IDTokenVerifier, logger *slog.Logger) *authenticator {
	return &authenticator{oauth: oauthConfig, verifier: verifier, store: newSessionStore(), logger: logger}
}

func (a *authenticator) Register(mux *http.ServeMux, proxy http.Handler) {
	mux.HandleFunc("/auth/login", a.login)
	mux.HandleFunc("/auth/callback", a.callback)
	mux.HandleFunc("/auth/logout", a.logout)
	mux.Handle("/", a.requireSession(proxy))
}

func (a *authenticator) login(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		a.fail(w, "could not create login state", http.StatusInternalServerError, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "hayel_oauth_state", Value: state, Path: "/auth", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.Redirect(w, r, a.oauth.AuthCodeURL(state), http.StatusFound)
}

func (a *authenticator) callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("hayel_oauth_state")
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		a.fail(w, "invalid OAuth state", http.StatusBadRequest, err)
		return
	}
	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		a.fail(w, "could not exchange authorization code", http.StatusUnauthorized, err)
		return
	}
	idTokenValue, ok := token.Extra("id_token").(string)
	if !ok || idTokenValue == "" {
		a.fail(w, "OIDC provider did not return an ID token", http.StatusUnauthorized, nil)
		return
	}
	idToken, err := a.verifier.Verify(r.Context(), idTokenValue)
	if err != nil {
		a.fail(w, "could not verify ID token", http.StatusUnauthorized, err)
		return
	}
	// Any identity successfully verified by the configured OIDC provider is
	// authorized. There is intentionally no second, local user allowlist.
	session, err := randomToken()
	if err != nil {
		a.fail(w, "could not create session", http.StatusInternalServerError, err)
		return
	}
	a.store.put(session, idToken.Subject)
	http.SetCookie(w, &http.Cookie{Name: "hayel_session", Value: session, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 60 * 60})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *authenticator) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("hayel_session"); err == nil {
		a.store.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "hayel_session", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *authenticator) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("hayel_session")
		if err != nil || !a.store.valid(cookie.Value) {
			if strings.HasPrefix(r.Header.Get("Accept"), "text/html") {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *authenticator) fail(w http.ResponseWriter, message string, status int, err error) {
	if err != nil {
		a.logger.Error(message, "error", err)
	} else {
		a.logger.Error(message)
	}
	http.Error(w, message, status)
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
