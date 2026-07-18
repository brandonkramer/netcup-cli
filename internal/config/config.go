package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	DefaultBaseURL     = "https://www.servercontrolpanel.de"
	DefaultProfile     = "default"
	DefaultWaitTimeout = 30 * time.Minute
	DefaultPollInterval = 2 * time.Second
	OIDCIssuer         = "https://www.servercontrolpanel.de/realms/scp"
	ClientID           = "scp"
)

type Config struct {
	BaseURL      string        `koanf:"base_url"`
	Profile      string        `koanf:"profile"`
	Format       string        `koanf:"format"`
	WaitTimeout  time.Duration `koanf:"wait_timeout"`
	PollInterval time.Duration `koanf:"poll_interval"`
	CacheEnabled bool          `koanf:"cache"`
	ConfigDir    string
}

func DefaultConfigDir() string {
	if v := os.Getenv("NETCUP_CONFIG_DIR"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "netcup")
	}
	return filepath.Join(home, ".config", "netcup")
}

func Load(profile, configDir string) (*Config, error) {
	if configDir == "" {
		configDir = DefaultConfigDir()
	}
	if profile == "" {
		profile = os.Getenv("NETCUP_PROFILE")
	}
	if profile == "" {
		profile = DefaultProfile
	}

	cfg := &Config{
		BaseURL:      DefaultBaseURL,
		Profile:      profile,
		Format:       "",
		WaitTimeout:  DefaultWaitTimeout,
		PollInterval: DefaultPollInterval,
		CacheEnabled: true,
		ConfigDir:    configDir,
	}

	k := koanf.New(".")
	path := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
	}
	_ = k.Load(env.Provider("NETCUP_", ".", func(s string) string {
		switch s {
		case "NETCUP_BASE_URL":
			return "base_url"
		case "NETCUP_FORMAT":
			return "format"
		case "NETCUP_PROFILE":
			return "profile"
		default:
			return ""
		}
	}), nil)

	if v := k.String("base_url"); v != "" {
		cfg.BaseURL = v
	}
	if v := k.String("format"); v != "" {
		cfg.Format = v
	}
	if v := k.String("profile"); v != "" && os.Getenv("NETCUP_PROFILE") == "" && profile == DefaultProfile {
		cfg.Profile = v
	}
	if os.Getenv("NETCUP_BASE_URL") != "" {
		cfg.BaseURL = os.Getenv("NETCUP_BASE_URL")
	}
	if os.Getenv("NETCUP_FORMAT") != "" {
		cfg.Format = os.Getenv("NETCUP_FORMAT")
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return cfg, nil
}

func (c *Config) CredentialsPath() string {
	return filepath.Join(c.ConfigDir, fmt.Sprintf("credentials-%s.json", c.Profile))
}

func (c *Config) CacheDir() string {
	return filepath.Join(c.ConfigDir, "cache", c.Profile)
}

func (c *Config) OpenAPIPath() string {
	if v := os.Getenv("NETCUP_OPENAPI"); v != "" {
		return v
	}
	return filepath.Join(c.ConfigDir, "openapi.json")
}

func (c *Config) APIRoot() string {
	return c.BaseURL + "/scp-core"
}

func (c *Config) DeviceAuthURL() string {
	return OIDCIssuer + "/protocol/openid-connect/auth/device"
}

func (c *Config) TokenURL() string {
	return OIDCIssuer + "/protocol/openid-connect/token"
}
