package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

var idfreeTokenPattern = regexp.MustCompile(`<meta name="x-token" content="([^"]+)"`)

type idfreeProvider struct {
	id      string
	name    string
	baseURL string
	client  *http.Client
}

type idfreeRawAccount struct {
	ID               int    `json:"id"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Message          string `json:"message"`
	LastCheck        string `json:"last_check"`
	LastCheckSuccess int    `json:"last_check_success"`
	RegionDisplay    string `json:"region_display"`
	Status           bool   `json:"status"`
}

func init() {
	Register("idfree", newIDFreeProvider)
}

func newIDFreeProvider(cfg Config) (Provider, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("idfree provider %q requires url", cfg.ID)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	return &idfreeProvider{
		id:      cfg.ID,
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.URL, "/"),
		client: &http.Client{
			Timeout: optionTimeout(cfg, 20*time.Second),
			Jar:     jar,
		},
	}, nil
}

func (p *idfreeProvider) ID() string {
	return p.id
}

func (p *idfreeProvider) Name() string {
	return p.name
}

func (p *idfreeProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	token, err := p.establishSession(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.verifySession(ctx, token); err != nil {
		return nil, err
	}

	body, err := p.fetchAccounts(ctx, token)
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(body), "INVALID_BROWSER") {
		return nil, fmt.Errorf("idfree rejected browser headers")
	}

	var raw []idfreeRawAccount
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse idfree response: %w", err)
	}

	accounts := make([]model.Account, 0, len(raw))
	for _, item := range raw {
		status := model.StatusUnavailable
		label := "异常"
		message := strings.TrimSpace(item.Message)
		if item.Status {
			status = model.StatusAvailable
			label = "可用"
			if message == "" {
				message = "检测正常，可登录 App Store"
			}
		} else if message == "" {
			message = "账号异常，请勿使用"
		}
		accountID := fmt.Sprintf("%s:%d", p.id, item.ID)
		accounts = append(accounts, model.Account{
			ID:            accountID,
			Channel:       p.id,
			ChannelName:   p.name,
			Country:       strings.TrimSpace(item.RegionDisplay),
			Username:      strings.TrimSpace(item.Username),
			Password:      item.Password,
			Status:        status,
			StatusMessage: message,
			StatusLabel:   label,
			UpdatedAt:     item.LastCheck,
			SourceURL:     p.baseURL + "/",
		})
	}
	return accounts, nil
}

func (p *idfreeProvider) establishSession(ctx context.Context) (string, error) {
	headers := p.baseHeaders()
	if _, err := fetchBody(ctx, p.client, http.MethodGet, p.baseURL+"/", headers, nil); err != nil {
		return "", fmt.Errorf("initialize idfree session: %w", err)
	}
	body, err := fetchBody(ctx, p.client, http.MethodGet, p.baseURL+"/", headers, nil)
	if err != nil {
		return "", fmt.Errorf("load idfree page: %w", err)
	}
	match := idfreeTokenPattern.FindSubmatch(body)
	if len(match) < 2 {
		return "", fmt.Errorf("idfree x-token not found in page")
	}
	token := strings.TrimSpace(string(match[1]))
	if token == "" {
		return "", fmt.Errorf("idfree x-token is empty")
	}
	return token, nil
}

func (p *idfreeProvider) verifySession(ctx context.Context, token string) error {
	headers := p.baseHeaders()
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("X-Token", token)
	if _, err := fetchBody(ctx, p.client, http.MethodPost, p.baseURL+"/api/session_verify.php", headers, strings.NewReader("")); err != nil {
		return fmt.Errorf("verify idfree session: %w", err)
	}
	return nil
}

func (p *idfreeProvider) fetchAccounts(ctx context.Context, token string) ([]byte, error) {
	headers := p.baseHeaders()
	headers.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	headers.Set("Sec-Fetch-Dest", "empty")
	headers.Set("Sec-Fetch-Mode", "cors")
	headers.Set("Sec-Fetch-Site", "same-origin")
	headers.Set("X-Token", token)
	return fetchBody(ctx, p.client, http.MethodGet, p.baseURL+"/api/accounts.php", headers, nil)
}

func (p *idfreeProvider) baseHeaders() http.Header {
	headers := http.Header{}
	headers.Set("User-Agent", defaultBrowserUserAgent)
	headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	headers.Set("Referer", p.baseURL+"/")
	headers.Set("Origin", p.baseURL)
	headers.Set("X-Requested-With", "XMLHttpRequest")
	headers.Set("Cache-Control", "no-cache")
	return headers
}
