package provider

import (
	"context"
	"fmt"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

type Provider interface {
	ID() string
	Name() string
	Fetch(ctx context.Context) ([]model.Account, error)
}

type Config struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	URL     string         `json:"url"`
	Enabled bool           `json:"enabled"`
	Options map[string]any `json:"options,omitempty"`
}

type Factory func(cfg Config) (Provider, error)

var factories = map[string]Factory{}

func Register(providerType string, factory Factory) {
	factories[providerType] = factory
}

func Build(cfg Config) (Provider, error) {
	factory, ok := factories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown provider type %q", cfg.Type)
	}
	return factory(cfg)
}
