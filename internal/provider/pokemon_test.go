package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPokemonProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://web4.52pokemon.cc/" {
			t.Errorf("Referer = %q, want https://web4.52pokemon.cc/", r.Header.Get("Referer"))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte({
			"code": 200,
			"msg": "获取成功",
			"status": true,
			"accounts": [
				{
					"id": 11,
					"username": "user1@163.com",
					"password": "pass1",
					"message": "正常",
					"last_check": "2026-08-29 01:10:14",
					"region_display": "美国",
					"status": true
				},
				{
					"id": 12,
					"username": "user2@outlook.com",
					"password": "pass2",
					"message": "请求被限流",
					"last_check": "2026-08-29 01:12:00",
					"region_display": "美国",
					"status": true
				}
			]
		}))
	}))
	defer server.Close()

	p, err := Build(Config{
		ID:   "pokemon_test",
		Type: "pokemon",
		Name: "宝可梦共享",
		URL:  server.URL,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	accounts, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(accounts) != 2 {
		t.Fatalf("len(accounts) = %d, want 2", len(accounts))
	}

	acc0 := accounts[0]
	if acc0.Username != "user1@163.com" || acc0.Status != "available" || !acc0.Shadowrocket {
		t.Errorf("acc0 = %+v", acc0)
	}

	acc1 := accounts[1]
	if acc1.Username != "user2@outlook.com" || acc1.Status != "unavailable" || !acc1.Shadowrocket {
		t.Errorf("acc1 = %+v", acc1)
	}
}
