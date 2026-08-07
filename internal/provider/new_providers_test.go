package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestParseIOSAppText(t *testing.T) {
	raw, err := parseIOSAppText("账号: demo@example.com\n密码: abc123\n状态: 账号可用\n")
	if err != nil {
		t.Fatalf("parseIOSAppText() error = %v", err)
	}
	if raw.Account != "demo@example.com" || raw.Password != "abc123" || raw.Status != "账号可用" {
		t.Fatalf("unexpected parse result: %+v", raw)
	}
}
