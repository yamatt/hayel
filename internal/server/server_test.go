package server

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestServerNewRejectsBadIssuer(t *testing.T) {
	cfg := Config{
		Listen:     ":8080",
		OIDCIssuer: "http://bad-issuer.example.com",
	}
	_, err := New(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected New to fail when the OIDC issuer cannot be discovered")
	}
	if !strings.Contains(err.Error(), "discover OIDC provider") {
		t.Fatalf("error = %q, want to mention OIDC discovery", err)
	}
}

func TestConfigValidateRejectsMissingIssuer(t *testing.T) {
	cfg := Config{Listen: ":8080"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation to fail when the issuer is missing")
	}
	if !strings.Contains(err.Error(), "OIDC configuration is incomplete") {
		t.Fatalf("error = %q, want OIDC validation failure", err)
	}
}
