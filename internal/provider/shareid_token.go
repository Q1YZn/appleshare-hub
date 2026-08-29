package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

const shareIDTokenDefaultUserAgent = "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Mobile Safari/537.36"

var emailValidCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9@.]`)

type shareIDTokenProvider struct {
	id            string
	name          string
	accountsURL   string
	tokenURL      string
	sessionCookie string
	client        *http.Client
	userAgent     string
}

type shareIDTokenResponse struct {
	Code    int    `json:"code"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

type shareIDRawAccount struct {
	Check    int64  `json:"check"`
	Country  string `json:"country"`
	Msg      string `json:"msg"`
	Password string `json:"password"`
	Status   int    `json:"status"`
	Time     string `json:"time"`
	User     int64  `json:"user"`
	Username string `json:"username"`
}

type shareIDAccountsResponse struct {
	Code int                 `json:"code"`
	Data []shareIDRawAccount `json:"data"`
}

func init() {
	Register("shareid_token", newShareIDTokenProvider)
}

func newShareIDTokenProvider(cfg Config) (Provider, error) {
	accountsURL, tokenURL := resolveShareIDURLs(cfg)
	if accountsURL == "" {
		return nil, fmt.Errorf("shareid_token provider %q requires url or options.accounts_url", cfg.ID)
	}

	timeout := 15 * time.Second
	if v, ok := cfg.Options["request_timeout_seconds"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}

	return &shareIDTokenProvider{
		id:            cfg.ID,
		name:          cfg.Name,
		accountsURL:   accountsURL,
		tokenURL:      tokenURL,
		sessionCookie: sessionCookie(cfg),
		client:        &http.Client{Timeout: timeout},
		userAgent:     shareIDTokenDefaultUserAgent,
	}, nil
}

func sessionCookie(cfg Config) string {
	if raw, ok := cfg.Options["session_cookie"].(string); ok {
		if value := strings.TrimSpace(raw); value != "" {
			return value
		}
	}
	return strings.TrimSpace(os.Getenv("SHAREID_SESSION_COOKIE"))
}

func resolveShareIDURLs(cfg Config) (accountsURL, tokenURL string) {
	if raw, ok := cfg.Options["accounts_url"].(string); ok && strings.TrimSpace(raw) != "" {
		accountsURL = strings.TrimSpace(raw)
	} else {
		accountsURL = strings.TrimSpace(cfg.URL)
	}

	if raw, ok := cfg.Options["token_url"].(string); ok && strings.TrimSpace(raw) != "" {
		tokenURL = strings.TrimSpace(raw)
	} else if accountsURL != "" {
		if strings.HasSuffix(accountsURL, "/b.php") {
			tokenURL = strings.TrimSuffix(accountsURL, "/b.php") + "/a.php"
		} else if strings.Contains(accountsURL, "/b.php?") {
			tokenURL = strings.Replace(accountsURL, "/b.php?", "/a.php?", 1)
		} else if strings.HasSuffix(accountsURL, "b.php") {
			tokenURL = strings.TrimSuffix(accountsURL, "b.php") + "a.php"
		} else if strings.HasSuffix(accountsURL, "/") {
			tokenURL = accountsURL + "a.php"
		} else {
			tokenURL = accountsURL + "/a.php"
		}
	}
	return accountsURL, tokenURL
}

func cleanShareIDUsername(raw string) string {
	s := strings.ReplaceAll(raw, `\@`, "@")
	cleaned := emailValidCharsRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(cleaned)
}

func (p *shareIDTokenProvider) ID() string {
	return p.id
}

func (p *shareIDTokenProvider) Name() string {
	return p.name
}

func (p *shareIDTokenProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	token, cookies, err := p.fetchToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch token: %w", err)
	}

	accounts, err := p.fetchAccounts(ctx, token, cookies)
	if err != nil {
		return nil, fmt.Errorf("fetch accounts: %w", err)
	}

	return accounts, nil
}

func (p *shareIDTokenProvider) fetchToken(ctx context.Context) (string, []*http.Cookie, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	if p.sessionCookie != "" {
		req.Header.Set("Cookie", p.sessionCookie)
	}

	if parsed, err := url.Parse(p.tokenURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		req.Header.Set("Origin", baseURL)
		req.Header.Set("Referer", baseURL+"/user/index/share")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("do token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", nil, fmt.Errorf("read token response: %w", err)
	}

	var res shareIDTokenResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", nil, fmt.Errorf("parse token response: %w", err)
	}

	if res.Code != 200 {
		msg := res.Message
		if msg == "" {
			msg = fmt.Sprintf("code %d", res.Code)
		}
		return "", nil, fmt.Errorf("token endpoint error: %s", msg)
	}

	token := strings.TrimSpace(res.Token)
	if token == "" {
		return "", nil, fmt.Errorf("token endpoint returned empty token")
	}

	return token, resp.Cookies(), nil
}

func (p *shareIDTokenProvider) fetchAccounts(ctx context.Context, token string, tokenCookies []*http.Cookie) ([]model.Account, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.accountsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create accounts request: %w", err)
	}

	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "*/*")

	if parsed, err := url.Parse(p.accountsURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		req.Header.Set("Referer", baseURL+"/user/index/share")
	}

	cookieParts := []string{fmt.Sprintf("Grid=%s", token)}
	if p.sessionCookie != "" {
		cookieParts = append(cookieParts, p.sessionCookie)
	} else {
		for _, cookie := range tokenCookies {
			if cookie.Name != "Grid" {
				cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
			}
		}
	}
	req.Header.Set("Cookie", strings.Join(cookieParts, "; "))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do accounts request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("accounts endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read accounts response: %w", err)
	}

	var res shareIDAccountsResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("parse accounts response: %w", err)
	}

	if res.Code != 200 {
		return nil, fmt.Errorf("accounts endpoint returned code %d", res.Code)
	}

	accounts := make([]model.Account, 0, len(res.Data))
	seen := make(map[string]struct{})

	for _, item := range res.Data {
		cleanedUsername := cleanShareIDUsername(item.Username)
		if cleanedUsername == "" {
			continue
		}

		if _, exists := seen[cleanedUsername]; exists {
			continue
		}
		seen[cleanedUsername] = struct{}{}

		status, label, message := mapShareIDTokenStatus(item.Status)
		if item.Msg != "" && (message == "" || status == model.StatusAvailable) {
			if status == model.StatusAvailable {
				message = fmt.Sprintf("%s，可登录 App Store", item.Msg)
			} else {
				message = item.Msg
			}
		}
		if label == "" {
			label = message
		}

		accounts = append(accounts, model.Account{
			ID:            fmt.Sprintf("%s:%s", p.id, cleanedUsername),
			Channel:       p.id,
			ChannelName:   p.name,
			Country:       item.Country,
			Username:      cleanedUsername,
			Password:      item.Password,
			Status:        status,
			StatusMessage: message,
			StatusLabel:   label,
			RawStatus:     item.Status,
			UpdatedAt:     item.Time,
			SourceURL:     p.accountsURL,
			Shadowrocket:  false,
		})
	}

	return accounts, nil
}

func mapShareIDTokenStatus(raw int) (model.Status, string, string) {
	switch raw {
	case 1:
		return model.StatusAvailable, "可用", "检测正常，可登录 App Store"
	case 0:
		return model.StatusChecking, "检测中", "账号正在检测，请稍后刷新"
	case 2:
		return model.StatusUnavailable, "异常", "账号异常，请勿使用"
	case 3:
		return model.StatusPending, "待检测", "账号等待检测"
	default:
		return model.StatusUnknown, "未知", "未知状态"
	}
}
