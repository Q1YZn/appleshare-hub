package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/provider"
)

type ServerConfig struct {
	Port                  int `json:"port"`
	CacheTTLSeconds       int `json:"cache_ttl_seconds"`
	RequestTimeoutSeconds int `json:"request_timeout_seconds"`
}

type Config struct {
	Server    ServerConfig      `json:"server"`
	Providers []provider.Config `json:"providers"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Port:                  8080,
			CacheTTLSeconds:       30,
			RequestTimeoutSeconds: 15,
		},
		Providers: []provider.Config{
			{
				ID:      "sha_cx_01",
				Type:    "sha_cx",
				Name:    "渠道 A（sha.cx）",
				URL:     "https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd",
				Enabled: true,
			},
			{
				ID:      "sha_cx_02",
				Type:    "sha_cx",
				Name:    "渠道 B（sha.cx）",
				URL:     "https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f",
				Enabled: true,
			},
		},
	}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.CacheTTLSeconds <= 0 {
		cfg.Server.CacheTTLSeconds = 30
	}
	if cfg.Server.RequestTimeoutSeconds <= 0 {
		cfg.Server.RequestTimeoutSeconds = 15
	}
	return cfg, nil
}

func (c Config) CacheTTL() time.Duration {
	return time.Duration(c.Server.CacheTTLSeconds) * time.Second
}

func (c Config) RequestTimeout() time.Duration {
	return time.Duration(c.Server.RequestTimeoutSeconds) * time.Second
}
