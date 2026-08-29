package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCleanShareIDUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Sample 1 from prompt",
			input:    `请删除uywjjt\@hao007.w所有中文in柾繍请不要登录设置否则手机会变砖`,
			expected: "uywjjt@hao007.win",
		},
		{
			name:     "Sample 2 from prompt",
			input:    `请删除xzndk9\@hao007.w所有中文i瘤擔n请不要登录设置否则手机会变砖`,
			expected: "xzndk9@hao007.win",
		},
		{
			name:     "Standard email",
			input:    "demo@example.com",
			expected: "demo@example.com",
		},
		{
			name:     "Escaped backslash email",
			input:    `test\.user\@gmail.com`,
			expected: "test.user@gmail.com",
		},
		{
			name:     "Empty / Chinese only",
			input:    "所有中文请不要登录设置",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanShareIDUsername(tt.input)
			if got != tt.expected {
				t.Errorf("cleanShareIDUsername(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShareIDTokenProviderFetch(t *testing.T) {
	var mu sync.Mutex
	var (
		sawPostToken   bool
		sawTokenCookie string
		sawGetBody     bool
		sawGridCookie  string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tools/shareid/a.php":
			if r.Method != http.MethodPost {
				http.Error(w, "method not post", http.StatusMethodNotAllowed)
				return
			}
			mu.Lock()
			sawPostToken = true
			sawTokenCookie = r.Header.Get("Cookie")
			mu.Unlock()

			http.SetCookie(w, &http.Cookie{Name: "server_name_session", Value: "session123"})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":200,"token":"grid_token_999","message":"Token generated successfully"}`))

		case "/tools/shareid/b.php":
			if r.Method != http.MethodGet {
				http.Error(w, "method not get", http.StatusMethodNotAllowed)
				return
			}
			mu.Lock()
			sawGetBody = true
			cookie, err := r.Cookie("Grid")
			if err == nil {
				sawGridCookie = cookie.Value
			}
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			resp := shareIDAccountsResponse{
				Code: 200,
				Data: []shareIDRawAccount{
					{
						Check:    1787929673,
						Country:  "美国",
						Msg:      "解锁成功",
						Password: "password123",
						Status:   1,
						Time:     "2026-08-28 23:07:53",
						User:     10329,
						Username: `请删除uywjjt\@hao007.w所有中文in柾繍请不要登录设置否则手机会变砖`,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p, err := Build(Config{
		ID:   "shareid_token_test",
		Type: "shareid_token",
		Name: "Test ShareID",
		URL:  server.URL + "/tools/shareid/b.php",
		Options: map[string]any{
			"session_cookie": "server_name_session=session123",
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(accounts) != 1 {
		t.Fatalf("Fetch() returned %d accounts, want 1", len(accounts))
	}

	acc := accounts[0]
	if acc.Username != "uywjjt@hao007.win" {
		t.Errorf("Username = %q, want %q", acc.Username, "uywjjt@hao007.win")
	}
	if acc.Password != "password123" {
		t.Errorf("Password = %q, want %q", acc.Password, "password123")
	}
	if acc.Country != "美国" {
		t.Errorf("Country = %q, want %q", acc.Country, "美国")
	}
	if acc.Status != "available" {
		t.Errorf("Status = %q, want available", acc.Status)
	}
	if acc.Shadowrocket {
		t.Errorf("Shadowrocket = true, want false")
	}

	mu.Lock()
	defer mu.Unlock()
	if !sawPostToken || !sawGetBody {
		t.Errorf("Flow incomplete: sawPostToken=%v sawGetBody=%v", sawPostToken, sawGetBody)
	}
	if sawGridCookie != "grid_token_999" {
		t.Errorf("Grid cookie = %q, want grid_token_999", sawGridCookie)
	}
	if sawTokenCookie != "server_name_session=session123" {
		t.Errorf("token Cookie = %q, want server_name_session=session123", sawTokenCookie)
	}
}

func TestResolveShareIDURLs(t *testing.T) {
	acc, tok := resolveShareIDURLs(Config{URL: "https://shop.bishojono1.com/tools/shareid/b.php"})
	if acc != "https://shop.bishojono1.com/tools/shareid/b.php" || tok != "https://shop.bishojono1.com/tools/shareid/a.php" {
		t.Errorf("resolveShareIDURLs() = (%q, %q)", acc, tok)
	}
}

