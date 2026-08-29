package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

type pokemonProvider struct {
	id     string
	name   string
	url    string
	client *http.Client
}

type pokemonResponse struct {
	Code     int              json:"code"
	Msg      string           json:"msg"
	Status   bool             json:"status"
	Accounts []pokemonAccount json:"accounts"
}

type pokemonAccount struct {
	ID            int    json:"id"
	Username      string json:"username"
	Password      string json:"password"
	Message       string json:"message"
	LastCheck     string json:"last_check"
	RegionDisplay string json:"region_display"
	Status        bool   json:"status"
}

func init() {
	Register("pokemon", newPokemonProvider)
}

func newPokemonProvider(cfg Config) (Provider, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		url = "https://appleid.52pokemon.cc/shareapi/MJFSqzxasI"
	}
	return &pokemonProvider{
		id:     cfg.ID,
		name:   cfg.Name,
		url:    url,
		client: newHTTPClient(optionTimeout(cfg, 15*time.Second)),
	}, nil
}

func (p *pokemonProvider) ID() string {
	return p.id
}

func (p *pokemonProvider) Name() string {
	return p.name
}

func (p *pokemonProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36",
		"Accept":     "application/json, text/plain, */*",
		"Referer":    "https://web4.52pokemon.cc/",
		"Origin":     "https://web4.52pokemon.cc",
	}
	body, err := fetchBody(ctx, p.client, http.MethodGet, p.url, headers, nil)
	if err != nil {
		return nil, err
	}

	var raw pokemonResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse pokemon response: %w", err)
	}
	if !raw.Status {
		return nil, fmt.Errorf("pokemon response status=false")
	}

	accounts := make([]model.Account, 0, len(raw.Accounts))
	seen := make(map[string]bool)
	for i, item := range raw.Accounts {
		username := strings.TrimSpace(item.Username)
		if username == "" || seen[username] {
			continue
		}
		seen[username] = true

		country := strings.TrimSpace(item.RegionDisplay)
		if country == "" {
			country = "美国"
		}

		message := strings.TrimSpace(item.Message)
		var status model.Status
		var label, statusMsg string

		if message == "正常" {
			status = model.StatusAvailable
			label = "可用"
			statusMsg = "检测正常，可登录 App Store"
		} else if message != "" {
			status = model.StatusUnavailable
			label = "异常"
			statusMsg = message
		} else if item.Status {
			status = model.StatusAvailable
			label = "可用"
			statusMsg = "检测正常，可登录 App Store"
		} else {
			status = model.StatusUnavailable
			label = "异常"
			statusMsg = "账号异常，请勿使用"
		}

		accID := fmt.Sprintf("%d", item.ID)
		if item.ID == 0 {
			accID = username
		}

		accounts = append(accounts, model.Account{
			ID:            fmt.Sprintf("%s:%s", p.id, accID),
			Channel:       p.id,
			ChannelName:   p.name,
			Country:       country,
			Username:      username,
			Password:      strings.TrimSpace(item.Password),
			Status:        status,
			StatusMessage: statusMsg,
			StatusLabel:   label,
			Priority:      i,
			UpdatedAt:     strings.TrimSpace(item.LastCheck),
			SourceURL:     p.url,
			Shadowrocket:  true,
		})
	}
	return accounts, nil
}
