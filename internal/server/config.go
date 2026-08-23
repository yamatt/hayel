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

type Config struct {
	Listen         string `koanf:"listen"`
	RepositoryRoot string `koanf:"repository_root"`
	OIDCIssuer     string `koanf:"oidc_issuer"`
	OIDCClientID   string `koanf:"oidc_client_id"`
	OIDCSecret     string `koanf:"oidc_client_secret"`
	OIDCRedirect   string `koanf:"oidc_redirect_url"`
}

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

	// 1. Defaults
	if err := k.Load(structs.Provider(structDefaults(), "koanf"), nil); err != nil {
		return Config{}, fmt.Errorf("load default configuration: %w", err)
	}

	// 2. TOML File
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
			return Config{}, fmt.Errorf("load configuration file %q: %w", configPath, err)
		}
	}

	// 3. Environment Variables (maps HAYEL_OIDC_ISSUER -> oidc_issuer)
	if err := k.Load(env.Provider("HAYEL_", ".", func(key string) string {
		return strings.ToLower(strings.TrimPrefix(key, "HAYEL_"))
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load environment configuration: %w", err)
	}

	// 4. Command-Line Flags (ONLY load flags that were EXPLICITLY set on the CLI)
	// Passing a key-transformation function normalizes "oidc-issuer" -> "oidc_issuer"
	if err := k.Load(posflag.ProviderWithValue(flags, ".", k, func(key string, value string) (string, interface{}) {
		// Convert hyphens to underscores so flags match struct tags (e.g. oidc-issuer -> oidc_issuer)
		normalKey := strings.ReplaceAll(key, "-", "_")

		// If flag wasn't explicitly changed on CLI, don't return it so it doesn't overwrite ENV vars!
		if !flags.Changed(key) {
			return "", nil
		}
		return normalKey, value
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load command-line configuration: %w", err)
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

// Relaxed validation: ONLY require Issuer for git-credential-oauth JWT verification
func (c Config) Validate() error {
	if c.OIDCIssuer == "" {
		return fmt.Errorf("OIDC configuration is incomplete: HAYEL_OIDC_ISSUER is required")
	}
	return nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
