package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

const shaCXDefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

var shaCXAdPattern = regexp.MustCompile(`(?s)ad\s*=\s*'([^']*)'`)

type shaCXProvider struct {
	id        string
	name      string
	urls      []string
	client    *http.Client
	userAgent string
}

type shaCXRawAccount struct {
	Check    int64  `json:"check"`
	Country  string `json:"country"`
	Msg      string `json:"msg"`
	Password string `json:"password"`
	Status   int    `json:"status"`
	Time     string `json:"time"`
	User     int    `json:"user"`
	Username string `json:"username"`
}

func init() {
	Register("sha_cx", newShaCXProvider)
}

func newShaCXProvider(cfg Config) (Provider, error) {
	urls := shaCXSourceURLs(cfg)
	if len(urls) == 0 {
		return nil, fmt.Errorf("sha_cx provider %q requires url or options.urls", cfg.ID)
	}
	timeout := 15 * time.Second
	if v, ok := cfg.Options["request_timeout_seconds"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}
	return &shaCXProvider{
		id:        cfg.ID,
		name:      cfg.Name,
		urls:      urls,
		client:    &http.Client{Timeout: timeout},
		userAgent: shaCXDefaultUserAgent,
	}, nil
}

func shaCXSourceURLs(cfg Config) []string {
	var urls []string
	if url := strings.TrimSpace(cfg.URL); url != "" {
		urls = append(urls, url)
	}
	if raw, ok := cfg.Options["urls"]; ok {
		switch items := raw.(type) {
		case []string:
			for _, item := range items {
				if url := strings.TrimSpace(item); url != "" {
					urls = append(urls, url)
				}
			}
		case []any:
			for _, item := range items {
				if url, ok := item.(string); ok {
					if url = strings.TrimSpace(url); url != "" {
						urls = append(urls, url)
					}
				}
			}
		}
	}
	seen := map[string]struct{}{}
	unique := urls[:0]
	for _, url := range urls {
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		unique = append(unique, url)
	}
	return unique
}

func (p *shaCXProvider) ID() string {
	return p.id
}

func (p *shaCXProvider) Name() string {
	return p.name
}

func (p *shaCXProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	type result struct {
		accounts []model.Account
		err      error
	}

	results := make([]result, len(p.urls))
	var wg sync.WaitGroup
	for i, url := range p.urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			accounts, err := p.fetchOne(ctx, url)
			results[i] = result{accounts: accounts, err: err}
		}(i, url)
	}
	wg.Wait()

	var accounts []model.Account
	var errs []string
	seen := map[string]struct{}{}
	for _, res := range results {
		if res.err != nil {
			errs = append(errs, res.err.Error())
			continue
		}
		for _, account := range res.accounts {
			if _, ok := seen[account.Username]; ok {
				continue
			}
			seen[account.Username] = struct{}{}
			accounts = append(accounts, account)
		}
	}
	if len(errs) > 0 {
		return accounts, fmt.Errorf("fetch %d/%d sha.cx sources: %s", len(errs), len(p.urls), strings.Join(errs, "; "))
	}
	return accounts, nil
}

func (p *shaCXProvider) fetchOne(ctx context.Context, url string) ([]model.Account, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	payload, err := extractShaCXPayload(body)
	if err != nil {
		return nil, err
	}

	var raw []shaCXRawAccount
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("parse account payload: %w", err)
	}

	accounts := make([]model.Account, 0, len(raw))
	for _, item := range raw {
		status, label, message := mapShaCXStatus(item.Status)
		if message == "" {
			message = item.Msg
		}
		if label == "" {
			label = message
		}
		accounts = append(accounts, model.Account{
			ID:            fmt.Sprintf("%s:%s", p.id, item.Username),
			Channel:       p.id,
			ChannelName:   p.name,
			Country:       item.Country,
			Username:      item.Username,
			Password:      item.Password,
			Status:        status,
			StatusMessage: message,
			StatusLabel:   label,
			RawStatus:     item.Status,
			UpdatedAt:     item.Time,
			SourceURL:     url,
			Shadowrocket:  true,
		})
	}
	return accounts, nil
}

func extractShaCXPayload(body []byte) ([]byte, error) {
	match := shaCXAdPattern.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("sha.cx payload not found in page")
	}
	payload := strings.TrimSpace(string(match[1]))
	if payload == "" {
		return nil, fmt.Errorf("sha.cx payload is empty")
	}
	return []byte(payload), nil
}

func mapShaCXStatus(raw int) (model.Status, string, string) {
	switch raw {
	case 0:
		return model.StatusChecking, "检测中", "账号正在检测，请稍后刷新"
	case 1:
		return model.StatusAvailable, "可用", "检测正常，可登录 App Store"
	case 2:
		return model.StatusUnavailable, "异常", "账号异常，请勿使用"
	case 3:
		return model.StatusPending, "待检测", "账号等待检测"
	default:
		return model.StatusUnknown, "未知", "未知状态"
	}
}
