package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

type fanqiangnanProvider struct {
	id     string
	name   string
	url    string
	client *http.Client
}

type fanqiangnanResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Accounts map[string][]fanqiangnanAccount `json:"accounts"`
	} `json:"data"`
}

type fanqiangnanAccount struct {
	ID         string `json:"id"`
	FullEmail  string `json:"fullEmail"`
	Password   string `json:"password"`
	Status     string `json:"status"`
	CheckTime  string `json:"checkTime"`
	Region     string `json:"region"`
	RegionName string `json:"regionName"`
}

func init() {
	Register("fanqiangnan", newFanqiangnanProvider)
}

func newFanqiangnanProvider(cfg Config) (Provider, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("fanqiangnan provider %q requires url", cfg.ID)
	}
	return &fanqiangnanProvider{
		id:     cfg.ID,
		name:   cfg.Name,
		url:    cfg.URL,
		client: newHTTPClient(optionTimeout(cfg, 15*time.Second)),
	}, nil
}

func (p *fanqiangnanProvider) ID() string {
	return p.id
}

func (p *fanqiangnanProvider) Name() string {
	return p.name
}

func (p *fanqiangnanProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	body, err := fetchBody(ctx, p.client, http.MethodGet, p.url, nil, nil)
	if err != nil {
		return nil, err
	}

	var raw fanqiangnanResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse fanqiangnan response: %w", err)
	}
	if !raw.Success {
		return nil, fmt.Errorf("fanqiangnan response success=false")
	}

	groups := make([]string, 0, len(raw.Data.Accounts))
	for group := range raw.Data.Accounts {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	accounts := make([]model.Account, 0, 32)
	for _, group := range groups {
		for _, item := range raw.Data.Accounts[group] {
			status, label, message := mapTextStatus(item.Status, "上游检测正常", "上游检测异常", "上游暂未提供状态")
			country := strings.TrimSpace(item.RegionName)
			if country == "" {
				country = strings.TrimSpace(item.Region)
			}
			accounts = append(accounts, model.Account{
				ID:            fmt.Sprintf("%s:%s:%s", p.id, group, item.ID),
				Channel:       p.id,
				ChannelName:   p.name,
				Country:       country,
				Username:      strings.TrimSpace(item.FullEmail),
				Password:      item.Password,
				Status:        status,
				StatusMessage: message,
				StatusLabel:   label,
				UpdatedAt:     item.CheckTime,
				SourceURL:     p.url,
				Shadowrocket:  false,
			})
		}
	}
	return accounts, nil
}

func mapTextStatus(raw, normalMessage, abnormalMessage, unknownMessage string) (model.Status, string, string) {
	switch {
	case strings.Contains(raw, "正常"):
		return model.StatusAvailable, "可用", normalMessage
	case strings.Contains(raw, "异常") || strings.Contains(raw, "失效"):
		return model.StatusUnavailable, "异常", abnormalMessage
	default:
		return model.StatusPending, "待检测", unknownMessage
	}
}
