package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Q1YZn/appleshare-hub/internal/model"
)

func TestUnicornKnowledgeProviderFetchJSON(t *testing.T) {
	var mu sync.Mutex
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		authHeader = r.Header.Get("Authorization")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"items": [
					{
						"username": "demo@example.com",
						"password": "abc123",
						"country": "美国",
						"status": "正常",
						"updated_at": "2026-08-12 10:00:00"
					},
					{
						"account": "skip-no-email",
						"password": "xyz789"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	p, err := newUnicornKnowledgeProvider(Config{
		ID:   "unicorn",
		Name: "独角兽",
		URL:  server.URL,
		Options: map[string]any{
			"token": "secret-token",
		},
	})
	if err != nil {
		t.Fatalf("newUnicornKnowledgeProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}
	account := accounts[0]
	if account.Username != "demo@example.com" || account.Password != "abc123" {
		t.Fatalf("unexpected account: %+v", account)
	}
	if account.Status != "available" || account.Country != "美国" || account.UpdatedAt != "2026-08-12 10:00:00" {
		t.Fatalf("unexpected account meta: %+v", account)
	}
	if !account.Shadowrocket {
		t.Fatal("account.shadowrocket = false, want true")
	}

	mu.Lock()
	defer mu.Unlock()
	if authHeader != "Bearer secret-token" {
		t.Fatalf("Authorization header = %q, want %q", authHeader, "Bearer secret-token")
	}
}

func TestUnicornKnowledgeProviderFetchText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("demo@example.com abc123\nsecond@example.com pwd456\n"))
	}))
	defer server.Close()

	p, err := newUnicornKnowledgeProvider(Config{ID: "unicorn", URL: server.URL})
	if err != nil {
		t.Fatalf("newUnicornKnowledgeProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("Fetch() returned %d accounts, want 2", len(accounts))
	}
	if accounts[0].Username != "demo@example.com" || accounts[0].Password != "abc123" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
	if accounts[0].Status != "pending" {
		t.Fatalf("unexpected status: %s", accounts[0].Status)
	}
}

func TestUnicornKnowledgeProviderFetchErrorHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"未登录或登陆已过期"}`))
	}))
	defer server.Close()

	p, err := newUnicornKnowledgeProvider(Config{ID: "unicorn", URL: server.URL})
	if err != nil {
		t.Fatalf("newUnicornKnowledgeProvider() error = %v", err)
	}
	_, err = p.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want login error hint")
	}
	if !strings.Contains(err.Error(), "未登录") {
		t.Fatalf("Fetch() error = %v, want login hint", err)
	}
}

func TestUnicornKnowledgeProviderFetchHTTP403Hint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"未登录或登陆已过期"}`))
	}))
	defer server.Close()

	p, err := newUnicornKnowledgeProvider(Config{ID: "unicorn", URL: server.URL})
	if err != nil {
		t.Fatalf("newUnicornKnowledgeProvider() error = %v", err)
	}
	_, err = p.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() error = nil, want HTTP 403 login hint")
	}
	if !strings.Contains(err.Error(), "HTTP 403") || !strings.Contains(err.Error(), "未登录") {
		t.Fatalf("Fetch() error = %v, want HTTP 403 + login hint", err)
	}
}

func TestMapUnicornStatus(t *testing.T) {
	cases := []struct {
		raw    string
		status string
	}{
		{"正常", "available"},
		{"可用", "available"},
		{"success", "available"},
		{"异常", "unavailable"},
		{"failed", "unavailable"},
		{"", "pending"},
		{"unknown", "pending"},
	}
	for _, tc := range cases {
		status, _, _ := mapUnicornStatus(tc.raw)
		if status != model.Status(tc.status) {
			t.Fatalf("mapUnicornStatus(%q) = %q, want %q", tc.raw, status, tc.status)
		}
	}
}

func TestParseUnicornAccountText(t *testing.T) {
	accounts := parseUnicornAccountText("账号: demo@example.com 密码: abc123\n其他行\nsecond@example.com pwd456\n")
	if len(accounts) != 2 {
		t.Fatalf("parseUnicornAccountText() returned %d accounts, want 2", len(accounts))
	}
	if accounts[0]["username"] != "demo@example.com" || accounts[0]["password"] != "abc123" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
	if !strings.Contains(accounts[1]["password"].(string), "pwd456") {
		t.Fatalf("unexpected second account: %+v", accounts[1])
	}
}
