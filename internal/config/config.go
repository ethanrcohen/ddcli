package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configFileName = ".ddcli.json"
	envAPIKey      = "DD_API_KEY"
	envAppKey      = "DD_APP_KEY"
	envSite        = "DD_SITE"
)

// Config holds Datadog authentication and site configuration.
type Config struct {
	APIKey string `json:"api_key"`
	AppKey string `json:"app_key"`
	Site   string `json:"site"` // e.g. "datadoghq.com", "datadoghq.eu", "us5.datadoghq.com"
}

// BaseURL returns the API base URL derived from the configured site.
func (c Config) BaseURL() string {
	site := c.Site
	if site == "" {
		site = "datadoghq.com"
	}
	return fmt.Sprintf("https://api.%s", site)
}

// Validate returns an error if required fields are missing.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("DD_API_KEY is not set (use `ddcli configure` or set the DD_API_KEY env var)")
	}
	if c.AppKey == "" {
		return errors.New("DD_APP_KEY is not set (use `ddcli configure` or set the DD_APP_KEY env var)")
	}
	return nil
}

// Load reads config from env vars first, then falls back to the config file.
// Env vars always take precedence.
func Load() (Config, error) {
	cfg := Config{}

	// Try config file first as defaults
	if fileCfg, err := loadFromFile(); err == nil {
		cfg = fileCfg
	}

	// Env vars override file config
	if v := os.Getenv(envAPIKey); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv(envAppKey); v != "" {
		cfg.AppKey = v
	}
	if v := os.Getenv(envSite); v != "" {
		cfg.Site = v
	}

	if cfg.Site == "" {
		cfg.Site = "datadoghq.com"
	}

	return cfg, nil
}

// Save writes the config to the user's home directory.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func loadFromFile() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configFileName), nil
}
