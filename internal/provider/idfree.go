package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

var (
	idfreeTokenPattern   = regexp.MustCompile(`<meta name="x-token" content="([^"]+)"`)
	idfreeSiteKeyPattern = regexp.MustCompile(`data-sitekey="([^"]+)"`)
)

type idfreeProvider struct {
	id             string
	name           string
	baseURL        string
	client         *http.Client
	captchaSolver  string
	captchaAPIKey  string
	captchaAPIURL  string
	captchaTimeout time.Duration
	proxyURL       string
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
	captchaSolver := strings.ToLower(strings.TrimSpace(optionString(cfg, "captcha_solver")))
	captchaAPIKey := strings.TrimSpace(optionString(cfg, "captcha_api_key"))
	captchaAPIURL := strings.TrimRight(strings.TrimSpace(optionString(cfg, "captcha_api_url")), "/")
	proxyURL := strings.TrimRight(strings.TrimSpace(optionString(cfg, "proxy_url")), "/")
	if value := strings.TrimSpace(os.Getenv("IDFREE_CAPTCHA_SOLVER")); value != "" {
		captchaSolver = strings.ToLower(value)
	}
	if value := strings.TrimSpace(os.Getenv("IDFREE_CAPTCHA_API_KEY")); value != "" {
		captchaAPIKey = value
	}
	if value := strings.TrimSpace(os.Getenv("IDFREE_PROXY_URL")); value != "" {
		proxyURL = strings.TrimRight(value, "/")
	}
	captchaTimeout := 30 * time.Second
	if v, ok := cfg.Options["captcha_timeout_seconds"].(float64); ok && v > 0 {
		captchaTimeout = time.Duration(v) * time.Second
	}
	return &idfreeProvider{
		id:             cfg.ID,
		name:           cfg.Name,
		baseURL:        strings.TrimRight(cfg.URL, "/"),
		captchaSolver:  captchaSolver,
		captchaAPIKey:  captchaAPIKey,
		captchaAPIURL:  captchaAPIURL,
		captchaTimeout: captchaTimeout,
		proxyURL:       proxyURL,
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
	token, siteKey, err := p.establishSession(ctx)
	if err != nil {
		return nil, err
	}
	if siteKey != "" {
		if err := p.verifyTurnstile(ctx, siteKey); err != nil {
			return nil, err
		}
	}
	sessionToken, err := p.verifySession(ctx, token)
	if err != nil {
		return nil, err
	}

	body, err := p.fetchAccounts(ctx, sessionToken)
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
			Shadowrocket:  false,
		})
	}
	return accounts, nil
}

func (p *idfreeProvider) establishSession(ctx context.Context) (string, string, error) {
	headers := p.baseHeaders()
	if _, err := fetchBody(ctx, p.client, http.MethodGet, p.requestURL("/"), headers, nil); err != nil {
		return "", "", fmt.Errorf("initialize idfree session: %w", err)
	}
	body, err := fetchBody(ctx, p.client, http.MethodGet, p.requestURL("/"), headers, nil)
	if err != nil {
		return "", "", fmt.Errorf("load idfree page: %w", err)
	}
	match := idfreeTokenPattern.FindSubmatch(body)
	if len(match) < 2 {
		return "", "", fmt.Errorf("idfree x-token not found in page")
	}
	token := strings.TrimSpace(string(match[1]))
	if token == "" {
		return "", "", fmt.Errorf("idfree x-token is empty")
	}
	siteKey := ""
	if siteMatch := idfreeSiteKeyPattern.FindSubmatch(body); len(siteMatch) > 1 {
		siteKey = strings.TrimSpace(string(siteMatch[1]))
	}
	return token, siteKey, nil
}

func (p *idfreeProvider) verifySession(ctx context.Context, token string) (string, error) {
	headers := p.baseHeaders()
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("X-Token", token)
	body, err := fetchBody(ctx, p.client, http.MethodPost, p.requestURL("/api/session_verify.php"), headers, strings.NewReader(""))
	if err != nil {
		return "", fmt.Errorf("verify idfree session: %w", err)
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return token, nil
	}
	var result struct {
		OK    bool   `json:"ok"`
		Token string `json:"token"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return "", fmt.Errorf("parse idfree session response: %w", err)
	}
	if !result.OK {
		if result.Error == "" {
			return "", fmt.Errorf("verify idfree session: unknown error")
		}
		return "", fmt.Errorf("verify idfree session: %s", result.Error)
	}
	if result.Token != "" {
		return result.Token, nil
	}
	return token, nil
}

func (p *idfreeProvider) verifyTurnstile(ctx context.Context, siteKey string) error {
	if p.captchaSolver == "" || p.captchaAPIKey == "" {
		return fmt.Errorf("idfree 上游已开启 Cloudflare Turnstile，需要在 options 配置 captcha_solver 和 captcha_api_key（或环境变量 IDFREE_CAPTCHA_API_KEY）后恢复")
	}

	var token string
	var err error
	switch p.captchaSolver {
	case "capsolver":
		token, err = p.solveCapsolverTurnstile(ctx, siteKey)
	case "2captcha":
		token, err = p.solve2CaptchaTurnstile(ctx, siteKey)
	default:
		return fmt.Errorf("idfree unsupported captcha_solver %q (supported: capsolver, 2captcha)", p.captchaSolver)
	}
	if err != nil {
		return fmt.Errorf("idfree solve turnstile: %w", err)
	}

	payload, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return fmt.Errorf("encode turnstile token: %w", err)
	}
	headers := p.baseHeaders()
	headers.Set("Content-Type", "application/json")
	if _, err := fetchBody(ctx, p.client, http.MethodPost, p.requestURL("/api/verify-turnstile.php"), headers, bytes.NewReader(payload)); err != nil {
		return fmt.Errorf("verify idfree turnstile: %w", err)
	}
	return nil
}

func (p *idfreeProvider) requestURL(path string) string {
	target := p.baseURL + path
	if p.proxyURL == "" {
		return target
	}
	return p.proxyURL + "/fetch?url=" + url.QueryEscape(target)
}

func (p *idfreeProvider) solverAPIURL(fallback string) string {
	if p.captchaAPIURL != "" {
		return p.captchaAPIURL
	}
	return fallback
}

func (p *idfreeProvider) solveCapsolverTurnstile(ctx context.Context, siteKey string) (string, error) {
	baseURL := p.solverAPIURL("https://api.capsolver.com")
	var created struct {
		TaskID           string `json:"taskId"`
		ErrorDescription string `json:"errorDescription"`
	}
	err := postJSON(ctx, p.client, baseURL+"/createTask", map[string]any{
		"clientKey": p.captchaAPIKey,
		"task": map[string]any{
			"type":       "AntiTurnstileTaskProxyLess",
			"websiteURL": p.baseURL + "/",
			"websiteKey": siteKey,
		},
	}, &created)
	if err != nil {
		return "", fmt.Errorf("createTask: %w", err)
	}
	if created.TaskID == "" {
		message := created.ErrorDescription
		if message == "" {
			message = "missing taskId"
		}
		return "", fmt.Errorf("createTask: %s", message)
	}

	deadline := time.Now().Add(p.captchaTimeout)
	for {
		var result struct {
			Status   string `json:"status"`
			Solution *struct {
				Token string `json:"token"`
			} `json:"solution"`
			ErrorDescription string `json:"errorDescription"`
		}
		err := postJSON(ctx, p.client, baseURL+"/getTaskResult", map[string]string{
			"clientKey": p.captchaAPIKey,
			"taskId":    created.TaskID,
		}, &result)
		if err != nil {
			return "", fmt.Errorf("getTaskResult: %w", err)
		}
		if result.Status == "ready" && result.Solution != nil && result.Solution.Token != "" {
			return result.Solution.Token, nil
		}
		if result.Status == "failed" {
			message := result.ErrorDescription
			if message == "" {
				message = "unknown error"
			}
			return "", fmt.Errorf("task failed: %s", message)
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return "", fmt.Errorf("turnstile solve timed out")
}

func (p *idfreeProvider) solve2CaptchaTurnstile(ctx context.Context, siteKey string) (string, error) {
	baseURL := p.solverAPIURL("https://2captcha.com")
	form := url.Values{}
	form.Set("key", p.captchaAPIKey)
	form.Set("method", "turnstile")
	form.Set("sitekey", siteKey)
	form.Set("pageurl", p.baseURL+"/")
	form.Set("json", "1")
	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("User-Agent", defaultBrowserUserAgent)
	createBody, err := fetchBody(ctx, p.client, http.MethodPost, baseURL+"/in.php", headers, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("in.php: %w", err)
	}
	var created struct {
		Status  int    `json:"status"`
		Request string `json:"request"`
	}
	if err := json.Unmarshal(createBody, &created); err != nil {
		return "", fmt.Errorf("parse in.php response: %w", err)
	}
	if created.Status != 1 || created.Request == "" {
		message := created.Request
		if message == "" {
			message = "unknown error"
		}
		return "", fmt.Errorf("in.php: %s", message)
	}

	deadline := time.Now().Add(p.captchaTimeout)
	for {
		poll := url.Values{}
		poll.Set("key", p.captchaAPIKey)
		poll.Set("action", "get")
		poll.Set("id", created.Request)
		poll.Set("json", "1")
		body, err := fetchBody(ctx, p.client, http.MethodGet, baseURL+"/res.php?"+poll.Encode(), headers, nil)
		if err != nil {
			return "", fmt.Errorf("res.php: %w", err)
		}
		var result struct {
			Status  int    `json:"status"`
			Request string `json:"request"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("parse res.php response: %w", err)
		}
		if result.Status == 1 && result.Request != "" {
			return result.Request, nil
		}
		if result.Status == 0 {
			code := strings.ToUpper(result.Request)
			if code != "CAPTCHA_NOT_READY" && code != "CAPCHA_NOT_READY" {
				return "", fmt.Errorf("task failed: %s", result.Request)
			}
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return "", fmt.Errorf("turnstile solve timed out")
}

func (p *idfreeProvider) fetchAccounts(ctx context.Context, token string) ([]byte, error) {
	headers := p.baseHeaders()
	headers.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	headers.Set("Sec-Fetch-Dest", "empty")
	headers.Set("Sec-Fetch-Mode", "cors")
	headers.Set("Sec-Fetch-Site", "same-origin")
	headers.Set("X-Token", token)
	return fetchBody(ctx, p.client, http.MethodGet, p.requestURL("/api/accounts.php"), headers, nil)
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
