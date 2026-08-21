package server

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// Config contains the runtime configuration for the Hayel gateway.
type Config struct {
	Listen         string `koanf:"listen"`
	RepositoryRoot string `koanf:"repository_root"`
	OIDCIssuer     string `koanf:"oidc_issuer"`
	OIDCClientID   string `koanf:"oidc_client_id"`
	OIDCSecret     string `koanf:"oidc_client_secret"`
	OIDCRedirect   string `koanf:"oidc_redirect_url"`
}

// ConfigFromFlags loads configuration with this precedence, from lowest to highest:
// defaults, TOML file, HAYEL_* environment variables, and command-line flags.
func ConfigFromFlags() (Config, error) {
	flags := pflag.NewFlagSet("hayel-server", pflag.ContinueOnError)
	configPath := envOr("HAYEL_CONFIG", "")
	flags.StringVar(&configPath, "config", configPath, "path to a TOML configuration file")
	flags.String("listen", ":8080", "address to listen on")
	flags.String("repository-root", "/repositories", "directory containing bare Git repositories")
	flags.String("oidc-issuer", "", "OIDC issuer URL")
	flags.String("oidc-client-id", "", "OIDC client ID")
	flags.String("oidc-client-secret", "", "OIDC client secret")
	flags.String("oidc-redirect-url", "", "OIDC callback URL")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	k := koanf.New(".")
	if err := k.Load(structs.Provider(structDefaults(), "koanf"), nil); err != nil {
		return Config{}, fmt.Errorf("load default configuration: %w", err)
	}
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
			return Config{}, fmt.Errorf("load configuration file %q: %w", configPath, err)
		}
	}
	if err := k.Load(env.Provider("HAYEL_", ".", func(key string) string {
		return strings.ToLower(strings.TrimPrefix(key, "HAYEL_"))
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load environment configuration: %w", err)
	}
	if err := k.Load(posflag.Provider(flags, "", k), nil); err != nil {
		return Config{}, fmt.Errorf("load command-line configuration: %w", err)
	}
	for _, key := range []string{
		"repository-root",
		"oidc-issuer",
		"oidc-client-id",
		"oidc-client-secret",
		"oidc-redirect-url",
	} {
		if k.Exists(key) {
			if err := k.Set(strings.ReplaceAll(key, "-", "_"), k.Get(key)); err != nil {
				return Config{}, fmt.Errorf("normalize command-line configuration: %w", err)
			}
		}
	}

	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf", FlatPaths: true}); err != nil {
		return Config{}, fmt.Errorf("decode configuration: %w", err)
	}
	return cfg, nil
}

func structDefaults() Config {
	return Config{Listen: ":8080", RepositoryRoot: "/repositories"}
}

func (c Config) Validate() error {
	if c.OIDCIssuer == "" || c.OIDCClientID == "" || c.OIDCSecret == "" || c.OIDCRedirect == "" {
		return errIncompleteOIDCConfig
	}
	return nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
