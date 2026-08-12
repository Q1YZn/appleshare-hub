package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestFanqiangnanProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"accounts": {
					"group1": [
						{
							"id": "1-1",
							"fullEmail": "demo@example.com",
							"password": "abc123",
							"status": "正常",
							"checkTime": "2026-08-07 01:59:50",
							"region": "US",
							"regionName": "美国"
						}
					],
					"group2": []
				}
			}
		}`))
	}))
	defer server.Close()

	p, err := newFanqiangnanProvider(Config{ID: "fanqiangnan", URL: server.URL})
	if err != nil {
		t.Fatalf("newFanqiangnanProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}
	if accounts[0].Username != "demo@example.com" || accounts[0].Password != "abc123" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
	if accounts[0].Status != "available" || accounts[0].Country != "美国" {
		t.Fatalf("unexpected account meta: %+v", accounts[0])
	}
}

func TestAppleidAPIProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"success": true,
			"count": 1,
			"data": [
				{
					"id": "abc123",
					"email": "demo@example.com",
					"password": "abc123",
					"region": "美国",
					"status": "正常",
					"source": "test",
					"timestamp": "2026-08-07T03:29:46.423Z"
				}
			]
		}`))
	}))
	defer server.Close()

	p, err := newAppleidAPIProvider(Config{ID: "appleid_api", URL: server.URL})
	if err != nil {
		t.Fatalf("newAppleidAPIProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}
	if accounts[0].Status != "available" || accounts[0].UpdatedAt != "2026-08-07T03:29:46.423Z" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
}

func TestIOSAppTextProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("类型:\n账号: demo@example.com\n密码: abc123\n检查时间:\n状态: 账号可用\n"))
	}))
	defer server.Close()

	p, err := newIOSAppTextProvider(Config{
		ID:   "iosapp",
		Type: "iosapp_text",
		URL:  server.URL,
	})
	if err != nil {
		t.Fatalf("newIOSAppTextProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}
	if accounts[0].Username != "demo@example.com" || accounts[0].Status != "available" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
	if !strings.Contains(accounts[0].StatusMessage, "未提供检查时间") {
		t.Fatalf("missing unverified hint: %+v", accounts[0])
	}
}

func TestIDFreeProviderTurnstileFetch(t *testing.T) {
	var mu sync.Mutex
	var (
		sawCreateTask   bool
		sawGetTask      bool
		sawVerifyTurn   bool
		sawSessionToken string
		sawAccounts     bool
	)

	solver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/createTask":
			mu.Lock()
			sawCreateTask = true
			mu.Unlock()
			_, _ = w.Write([]byte(`{"taskId":"task-123"}`))
		case "/getTaskResult":
			mu.Lock()
			sawGetTask = true
			mu.Unlock()
			_, _ = w.Write([]byte(`{"status":"ready","solution":{"token":"turnstile-solved"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer solver.Close()

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html>
				<meta name="x-token" content="page-token-123">
				<div class="cf-turnstile" data-sitekey="test-site-key"></div>
			</html>`))
		case "/api/verify-turnstile.php":
			var payload struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if payload.Token != "turnstile-solved" {
				http.Error(w, "bad token", http.StatusForbidden)
				return
			}
			mu.Lock()
			sawVerifyTurn = true
			mu.Unlock()
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/session_verify.php":
			if got := r.Header.Get("X-Token"); got != "page-token-123" {
				http.Error(w, "bad page token", http.StatusForbidden)
				return
			}
			mu.Lock()
			sawSessionToken = r.Header.Get("X-Token")
			mu.Unlock()
			_, _ = w.Write([]byte(`{"ok":true,"token":"session-token-456"}`))
		case "/api/accounts.php":
			if got := r.Header.Get("X-Token"); got != "session-token-456" {
				http.Error(w, "bad session token", http.StatusForbidden)
				return
			}
			mu.Lock()
			sawAccounts = true
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"id": 1, "username": "demo@example.com", "password": "abc123", "message": "正常", "last_check": "2026-08-12 10:00:00", "last_check_success": 1, "region_display": "美国", "status": true}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	p, err := newIDFreeProvider(Config{
		ID:   "idfree",
		Name: "小优 ID",
		URL:  site.URL,
		Options: map[string]any{
			"captcha_solver":          "capsolver",
			"captcha_api_key":         "test-key",
			"captcha_api_url":         solver.URL,
			"captcha_timeout_seconds": float64(10),
		},
	})
	if err != nil {
		t.Fatalf("newIDFreeProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}
	if accounts[0].Username != "demo@example.com" || accounts[0].Status != "available" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}

	mu.Lock()
	defer mu.Unlock()
	if !sawCreateTask || !sawGetTask || !sawVerifyTurn || !sawAccounts {
		t.Fatalf("turnstile flow incomplete: createTask=%v getTaskResult=%v verifyTurnstile=%v accounts=%v",
			sawCreateTask, sawGetTask, sawVerifyTurn, sawAccounts)
	}
	if sawSessionToken != "page-token-123" {
		t.Fatalf("session_verify saw X-Token %q, want page-token-123", sawSessionToken)
	}
}

func TestIDFreeProviderTurnstileRequiresKey(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<meta name="x-token" content="page-token-123">
			<div class="cf-turnstile" data-sitekey="test-site-key"></div>
		</html>`))
	}))
	defer site.Close()

	p, err := newIDFreeProvider(Config{ID: "idfree", URL: site.URL})
	if err != nil {
		t.Fatalf("newIDFreeProvider() error = %v", err)
	}
	_, err = p.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want captcha configuration error")
	}
	if !strings.Contains(err.Error(), "captcha_solver") || !strings.Contains(err.Error(), "captcha_api_key") {
		t.Fatalf("Fetch() error = %v, want captcha configuration hint", err)
	}
}

func TestParseIOSAppText(t *testing.T) {
	raw, err := parseIOSAppText("账号: demo@example.com\n密码: abc123\n状态: 账号可用\n")
	if err != nil {
		t.Fatalf("parseIOSAppText() error = %v", err)
	}
	if raw.Account != "demo@example.com" || raw.Password != "abc123" || raw.Status != "账号可用" {
		t.Fatalf("unexpected parse result: %+v", raw)
	}
}
