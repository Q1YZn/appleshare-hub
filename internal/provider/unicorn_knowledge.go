package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

var (
	unicornEmailPattern         = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	unicornPasswordPattern      = regexp.MustCompile(`^([^\s|,，;；]+)`)
	unicornPasswordLabelPattern = regexp.MustCompile(`^(密码|password|pwd)([:：]|\s+)\s*`)
	unicornHTMLTagPattern       = regexp.MustCompile(`<[^>]+>`)
)

type unicornKnowledgeProvider struct {
	id     string
	name   string
	url    string
	token  string
	client *http.Client
}

func init() {
	Register("unicorn_knowledge", newUnicornKnowledgeProvider)
}

func newUnicornKnowledgeProvider(cfg Config) (Provider, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("unicorn_knowledge provider %q requires url", cfg.ID)
	}
	token := strings.TrimSpace(optionString(cfg, "token"))
	if value := strings.TrimSpace(os.Getenv("UNICORN_TOKEN")); value != "" {
		token = value
	}
	return &unicornKnowledgeProvider{
		id:     cfg.ID,
		name:   cfg.Name,
		url:    cfg.URL,
		token:  token,
		client: newHTTPClient(optionTimeout(cfg, 20*time.Second)),
	}, nil
}

func (p *unicornKnowledgeProvider) ID() string {
	return p.id
}

func (p *unicornKnowledgeProvider) Name() string {
	return p.name
}

func (p *unicornKnowledgeProvider) Fetch(ctx context.Context) ([]model.Account, error) {
	headers := http.Header{}
	headers.Set("User-Agent", defaultBrowserUserAgent)
	headers.Set("Accept", "application/json, text/plain, text/html, */*")
	if p.token != "" {
		auth := p.token
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			auth = "Bearer " + auth
		}
		headers.Set("Authorization", auth)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request source: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errBody any
		_ = json.Unmarshal(body, &errBody)
		if hint := unicornErrorHint(errBody); hint != "" {
			return nil, fmt.Errorf("source returned HTTP %d: %s", resp.StatusCode, hint)
		}
		return nil, fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}
	text := string(body)

	var parsed any
	_ = json.Unmarshal(body, &parsed)

	var rawAccounts []map[string]any
	if parsed != nil {
		collectUnicornAccountObjects(parsed, &rawAccounts, 0)
	}
	if len(rawAccounts) == 0 {
		for _, item := range parseUnicornAccountText(text) {
			rawAccounts = append(rawAccounts, item)
		}
	}
	if len(rawAccounts) == 0 {
		if hint := unicornErrorHint(parsed); hint != "" {
			return nil, fmt.Errorf("unicorn_knowledge: %s", hint)
		}
		return nil, fmt.Errorf("unicorn_knowledge: no accounts found in knowledge response")
	}

	accounts := make([]model.Account, 0, len(rawAccounts))
	seen := map[string]struct{}{}
	for index, item := range rawAccounts {
		username := firstUnicornString(item, "username", "email", "account", "user")
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		password := firstUnicornString(item, "password", "pass", "pwd")
		status, label, message := mapUnicornStatus(firstUnicornString(item, "status", "state", "result"))
		accounts = append(accounts, model.Account{
			ID:            fmt.Sprintf("%s:%s", p.id, username),
			Channel:       p.id,
			ChannelName:   p.name,
			Country:       firstUnicornString(item, "country", "region", "location"),
			Username:      username,
			Password:      password,
			Status:        status,
			StatusMessage: message,
			StatusLabel:   label,
			Priority:      index,
			UpdatedAt:     firstUnicornString(item, "updated_at", "time", "check_time"),
			SourceURL:     p.url,
			Shadowrocket:  true,
		})
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("unicorn_knowledge: no valid accounts in knowledge response")
	}
	return accounts, nil
}

func collectUnicornAccountObjects(value any, accounts *[]map[string]any, depth int) {
	if depth > 8 || value == nil {
		return
	}
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			collectUnicornAccountObjects(item, accounts, depth+1)
		}
	case map[string]any:
		username := firstUnicornString(v, "username", "email", "account", "user")
		password := firstUnicornString(v, "password", "pass", "pwd")
		if username != "" && password != "" && strings.Contains(username, "@") {
			*accounts = append(*accounts, v)
			return
		}
		for _, child := range v {
			collectUnicornAccountObjects(child, accounts, depth+1)
		}
	}
}

func firstUnicornString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if text, ok := value.(string); ok {
				if trimmed := strings.TrimSpace(text); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

func unicornErrorHint(parsed any) string {
	root, ok := parsed.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"message", "msg", "error", "reason"} {
		if value, ok := root[key]; ok {
			if text, ok := value.(string); ok {
				if hint := strings.TrimSpace(text); hint != "" {
					return hint
				}
			}
		}
	}
	return ""
}

func parseUnicornAccountText(content string) []map[string]any {
	var accounts []map[string]any
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(rawLine, "\u00a0", " "), "\r", ""))
		line = unicornHTMLTagPattern.ReplaceAllString(line, " ")
		for _, match := range unicornEmailPattern.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 2 {
				continue
			}
			username := line[match[0]:match[1]]
			tail := line[match[1]:]
			tail = strings.TrimLeft(tail, " :：|,，、")
			tail = unicornPasswordLabelPattern.ReplaceAllString(tail, "")
			passwordMatch := unicornPasswordPattern.FindStringSubmatch(tail)
			if len(passwordMatch) > 1 && len(passwordMatch[1]) >= 4 {
				accounts = append(accounts, map[string]any{
					"username": username,
					"password": passwordMatch[1],
				})
			}
		}
	}
	return accounts
}

func mapUnicornStatus(value string) (model.Status, string, string) {
	text := strings.ToLower(strings.TrimSpace(value))
	availableTokens := []string{"正常", "可用", "成功", "valid", "ok", "available", "success"}
	unavailableTokens := []string{"异常", "失效", "不可用", "已失效", "失败", "invalid", "unavailable", "failed", "error"}
	for _, token := range availableTokens {
		if strings.Contains(text, token) {
			return model.StatusAvailable, "可用", "知识库标记可用，建议使用前再确认"
		}
	}
	for _, token := range unavailableTokens {
		if strings.Contains(text, token) {
			return model.StatusUnavailable, "异常", "知识库标记异常，请勿使用"
		}
	}
	return model.StatusPending, "待确认", "知识库未提供明确状态"
}
