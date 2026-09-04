package server

import (
	"os"
	"testing"
)

func TestConfigValidateAndEnvOr(t *testing.T) {
	t.Setenv("HAYEL_CUSTOM", "from-env")
	if got := envOr("HAYEL_CUSTOM", "fallback"); got != "from-env" {
		t.Fatalf("envOr() = %q, want %q", got, "from-env")
	}
	if got := envOr("HAYEL_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("envOr() = %q, want %q", got, "fallback")
	}

	cfg := Config{Listen: ":8080"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation to fail when OIDC issuer is missing")
	}

	cfg.OIDCIssuer = "https://issuer.example.com"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to pass with issuer set: %v", err)
	}
}

func TestConfigFromFlagsUsesEnvAndFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Setenv("HAYEL_LISTEN", ":9090")
	t.Setenv("HAYEL_REPOSITORY_ROOT", "/tmp/repos")
	t.Setenv("HAYEL_OIDC_ISSUER", "https://issuer.example.com")
	t.Setenv("HAYEL_OIDC_CLIENT_ID", "env-client")
	t.Setenv("HAYEL_OIDC_CLIENT_SECRET", "env-secret")
	t.Setenv("HAYEL_OIDC_REDIRECT_URL", "https://git.example.com/auth/callback")
	os.Args = []string{"hayel-server", "--listen=:7070", "--repository-root=/tmp/flags"}

	cfg, err := ConfigFromFlags()
	if err != nil {
		t.Fatalf("ConfigFromFlags returned error: %v", err)
	}
	if cfg.Listen != ":7070" {
		t.Fatalf("Listen = %q, want %q", cfg.Listen, ":7070")
	}
	if cfg.RepositoryRoot != "/tmp/flags" {
		t.Fatalf("RepositoryRoot = %q, want %q", cfg.RepositoryRoot, "/tmp/flags")
	}
	if cfg.OIDCIssuer != "https://issuer.example.com" {
		t.Fatalf("OIDCIssuer = %q, want %q", cfg.OIDCIssuer, "https://issuer.example.com")
	}
	if cfg.OIDCClientID != "env-client" {
		t.Fatalf("OIDCClientID = %q, want %q", cfg.OIDCClientID, "env-client")
	}
}
