package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractShaCXPayload(t *testing.T) {
	html := []byte(`<html><body>var ad='[{"country":"美国","msg":"检测正常","password":"N\u0026tjHg3C","status":1,"time":"2026-08-07 00:50:47","username":"demo@example.com"}]';</body></html>`)
	payload, err := extractShaCXPayload(html)
	if err != nil {
		t.Fatalf("extractShaCXPayload() error = %v", err)
	}
	if got := string(payload); got == "" {
		t.Fatal("extractShaCXPayload() returned empty payload")
	}
}

func TestShaCXProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>ad='[{"country":"美国","msg":"检测正常","password":"abc123","status":1,"time":"2026-08-07 00:50:47","username":"demo@example.com"}]'</html>`))
	}))
	defer server.Close()

	cfg := Config{ID: "test", Type: "sha_cx", Name: "Test", URL: server.URL, Enabled: true}
	p, err := newShaCXProvider(cfg)
	if err != nil {
		t.Fatalf("newShaCXProvider() error = %v", err)
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
	if accounts[0].Status != "available" {
		t.Fatalf("unexpected status: %s", accounts[0].Status)
	}
}

func TestShaCXProviderFetchMultipleURLs(t *testing.T) {
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>ad='[{"username":"a@example.com","password":"aaa","status":1,"time":"2026-08-07 00:50:47"}]'</html>`))
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>ad='[{"username":"b@example.com","password":"bbb","status":1,"time":"2026-08-07 00:50:47"}]'</html>`))
	}))
	defer serverB.Close()

	cfg := Config{
		ID:   "test",
		Type: "sha_cx",
		Name: "Test",
		Options: map[string]any{
			"urls": []any{serverA.URL, serverB.URL},
		},
	}
	p, err := newShaCXProvider(cfg)
	if err != nil {
		t.Fatalf("newShaCXProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("Fetch() returned %d accounts, want 2", len(accounts))
	}
	seen := map[string]bool{}
	for _, account := range accounts {
		seen[account.Username] = true
		if account.Channel != "test" {
			t.Fatalf("unexpected channel: %s", account.Channel)
		}
	}
	if !seen["a@example.com"] || !seen["b@example.com"] {
		t.Fatalf("unexpected accounts: %+v", seen)
	}
}

func TestShaCXProviderFetchDeduplicatesByUsername(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>ad='[{"username":"same@example.com","password":"abc123","status":1,"time":"2026-08-07 00:50:47"}]'</html>`))
	}))
	defer server.Close()

	cfg := Config{
		ID:   "test",
		Type: "sha_cx",
		Name: "Test",
		Options: map[string]any{
			"urls": []string{server.URL, server.URL},
		},
	}
	p, err := newShaCXProvider(cfg)
	if err != nil {
		t.Fatalf("newShaCXProvider() error = %v", err)
	}
	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}
	if accounts[0].Username != "same@example.com" {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
}
