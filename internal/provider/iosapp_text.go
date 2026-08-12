package provider

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Q1YZn/appleshare-hub/internal/model"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type iosappTextProvider struct {
	id     string
	name   string
	urls   []string
	client *http.Client
}

type iosappTextAccount struct {
	Kind      string
	Account   string
	Password  string
	CheckTime string
	Status    string
}

func init() {
	Register("iosapp_text", newIOSAppTextProvider)
}

func newIOSAppTextProvider(cfg Config) (Provider, error) {
	urls := optionStringList(cfg, "urls")
	if len(urls) == 0 && strings.TrimSpace(cfg.URL) != "" {
		urls = []string{cfg.URL}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("iosapp_text provider %q requires options.urls", cfg.ID)
	}
	return &iosappTextProvider{
		id:     cfg.ID,
		name:   cfg.Name,
		urls:   urls,
		client: newHTTPClient(optionTimeout(cfg, 15*time.Second)),
	}, nil
}

func (p *iosappTextProvider) ID() string {
	return p.id
}

func (p *iosappTextProvider) Name() string {
	return p.name
}

func (p *iosappTextProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	accounts := make([]model.Account, 0, len(p.urls))
	errs := make([]string, 0, len(p.urls))
	for _, url := range p.urls {
		body, err := fetchBody(ctx, p.client, http.MethodGet, url, nil, nil)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		raw, err := parseIOSAppText(decodeMaybeGBK(body))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		status, label, message := mapIOSAppStatus(raw.Status, raw.CheckTime)
		accountID := strings.TrimSuffix(path.Base(url), path.Ext(url))
		if accountID == "" || accountID == "." {
			accountID = fmt.Sprintf("%d", len(accounts)+1)
		}
		accounts = append(accounts, model.Account{
			ID:            fmt.Sprintf("%s:%s", p.id, accountID),
			Channel:       p.id,
			ChannelName:   p.name,
			Username:      strings.TrimSpace(raw.Account),
			Password:      strings.TrimSpace(raw.Password),
			Status:        status,
			StatusMessage: message,
			StatusLabel:   label,
			Priority:      1,
			UpdatedAt:     strings.TrimSpace(raw.CheckTime),
			SourceURL:     url,
			Shadowrocket:  false,
		})
	}
	if len(errs) > 0 {
		return accounts, fmt.Errorf("iosapp_text partial failure: %s", strings.Join(errs, "; "))
	}
	return accounts, nil
}

func parseIOSAppText(content string) (iosappTextAccount, error) {
	var account iosappTextAccount
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			key, value, ok = strings.Cut(line, "：")
		}
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "类型":
			account.Kind = value
		case "账号":
			account.Account = value
		case "密码":
			account.Password = value
		case "检查时间":
			account.CheckTime = value
		case "状态":
			account.Status = value
		}
	}
	if account.Account == "" {
		return account, fmt.Errorf("账号字段缺失")
	}
	return account, nil
}

func mapIOSAppStatus(raw, checkTime string) (model.Status, string, string) {
	switch {
	case strings.Contains(raw, "可用"):
		label := "可用"
		message := "文本源标记可用"
		if strings.TrimSpace(checkTime) == "" {
			message = "文本源标记可用，但未提供检查时间，优先使用带检测时间的账号"
		}
		return model.StatusAvailable, label, message
	case strings.Contains(raw, "异常") || strings.Contains(raw, "失效"):
		return model.StatusUnavailable, "异常", "文本源标记异常，请勿使用"
	default:
		return model.StatusPending, "待检测", "文本源未提供明确状态"
	}
}

func optionStringList(cfg Config, key string) []string {
	raw, ok := cfg.Options[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				out = append(out, strings.TrimSpace(value))
			}
		}
		return out
	case string:
		if strings.TrimSpace(values) == "" {
			return nil
		}
		parts := strings.Split(values, ",")
		out := make([]string, 0, len(parts))
		for _, value := range parts {
			if strings.TrimSpace(value) != "" {
				out = append(out, strings.TrimSpace(value))
			}
		}
		return out
	default:
		return nil
	}
}

func decodeMaybeGBK(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(data)
	if err != nil {
		return string(data)
	}
	return string(decoded)
}
