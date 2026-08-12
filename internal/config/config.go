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
				Enabled: true,
				Options: map[string]any{
					"urls": []string{
						"https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd",
						"https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f",
					},
				},
			},
			{
				ID:      "fanqiangnan_01",
				Type:    "fanqiangnan",
				Name:    "翻墙男（fanqiangnan）",
				URL:     "https://fanqiangnan.com/data_sync.php",
				Enabled: true,
			},
			{
				ID:      "idfree_01",
				Type:    "idfree",
				Name:    "小优 ID（idfree）",
				URL:     "https://idfree.top/",
				Enabled: true,
				Options: map[string]any{
					"captcha_solver":          "capsolver",
					"captcha_api_key":         "",
					"proxy_url":               "",
					"captcha_timeout_seconds": 30,
				},
			},
			{
				ID:      "appleid_api_01",
				Type:    "appleid_api",
				Name:    "云码酷（appleid.uczyw.us）",
				URL:     "https://appleid.uczyw.us/api/accounts",
				Enabled: true,
			},
			{
				ID:      "appleid_api_02",
				Type:    "appleid_api",
				Name:    "云码酷备用（appleid2.uczyw.us）",
				URL:     "https://appleid2.uczyw.us/api/accounts",
				Enabled: false,
			},
			{
				ID:      "iosapp_text_01",
				Type:    "iosapp_text",
				Name:    "免费文本源（iosapp.icu，低优先级）",
				Enabled: true,
				Options: map[string]any{
					"urls": []string{
						"https://free.iosapp.icu/go-rod/1.txt",
						"https://free.iosapp.icu/go-rod/2.txt",
						"https://free.iosapp.icu/go-rod/3.txt",
					},
				},
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
