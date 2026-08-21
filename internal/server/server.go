package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var errIncompleteOIDCConfig = errors.New("OIDC configuration is incomplete")

// Server is the authenticated HTTP gateway in front of git-http-server.
type Server struct {
	logger *slog.Logger
	http   *http.Server
}

// New discovers the OIDC provider and constructs the gateway handler.
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	gitHTTP, err := newGitHTTPHandler(cfg.RepositoryRoot, logger)
	if err != nil {
		return nil, fmt.Errorf("configure git HTTP backend: %w", err)
	}

	auth := newAuthenticator(
		&oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.OIDCRedirect,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		logger,
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/hayel-config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Issuer   string `json:"issuer"`
			ClientID string `json:"client_id"`
		}{Issuer: cfg.OIDCIssuer, ClientID: cfg.OIDCClientID})
	})
	auth.Register(mux, gitHTTP)

	return &Server{
		logger: logger,
		http:   &http.Server{Addr: cfg.Listen, Handler: requestLogger(logger, mux)},
	}, nil
}

// ListenAndServe starts the gateway and blocks until the HTTP server exits.
func (s *Server) ListenAndServe() error {
	s.logger.Info("hayel-server listening", "address", s.http.Addr)
	return s.http.ListenAndServe()
}
