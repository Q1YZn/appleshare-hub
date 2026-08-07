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

type appleidAPIProvider struct {
	id     string
	name   string
	url    string
	client *http.Client
}

type appleidAPIResponse struct {
	Success bool                `json:"success"`
	Count   int                 `json:"count"`
	Data    []appleidAPIAccount `json:"data"`
}

type appleidAPIAccount struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Region   string `json:"region"`
	Status   string `json:"status"`
	Source   string `json:"source"`
	Time     string `json:"timestamp"`
}

func init() {
	Register("appleid_api", newAppleidAPIProvider)
}

func newAppleidAPIProvider(cfg Config) (Provider, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("appleid_api provider %q requires url", cfg.ID)
	}
	return &appleidAPIProvider{
		id:     cfg.ID,
		name:   cfg.Name,
		url:    cfg.URL,
		client: newHTTPClient(optionTimeout(cfg, 15*time.Second)),
	}, nil
}

func (p *appleidAPIProvider) ID() string {
	return p.id
}

func (p *appleidAPIProvider) Name() string {
	return p.name
}

func (p *appleidAPIProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	body, err := fetchBody(ctx, p.client, http.MethodGet, p.url, nil, nil)
	if err != nil {
		return nil, err
	}

	var raw appleidAPIResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse appleid_api response: %w", err)
	}
	if !raw.Success {
		return nil, fmt.Errorf("appleid_api response success=false")
	}

	accounts := make([]model.Account, 0, len(raw.Data))
	for _, item := range raw.Data {
		status, label, message := mapTextStatus(item.Status, "检测正常，可登录 App Store", "账号异常，请勿使用", "上游暂未提供状态")
		accountID := strings.TrimSpace(item.ID)
		if accountID == "" {
			accountID = strings.TrimSpace(item.Email)
		}
		accounts = append(accounts, model.Account{
			ID:            fmt.Sprintf("%s:%s", p.id, accountID),
			Channel:       p.id,
			ChannelName:   p.name,
			Country:       strings.TrimSpace(item.Region),
			Username:      strings.TrimSpace(item.Email),
			Password:      item.Password,
			Status:        status,
			StatusMessage: message,
			StatusLabel:   label,
			UpdatedAt:     item.Time,
			SourceURL:     p.url,
		})
	}
	return accounts, nil
}
