package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

func optionTimeout(cfg Config, fallback time.Duration) time.Duration {
	if v, ok := cfg.Options["request_timeout_seconds"].(float64); ok && v > 0 {
		return time.Duration(v) * time.Second
	}
	return fallback
}

func optionString(cfg Config, name string) string {
	if cfg.Options == nil {
		return ""
	}
	value, ok := cfg.Options[name]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func postJSON(ctx context.Context, client *http.Client, url string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode json payload: %w", err)
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", defaultBrowserUserAgent)
	body, err := fetchBody(ctx, client, http.MethodPost, url, headers, bytes.NewReader(data))
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse json response: %w", err)
	}
	return nil
}

func fetchBody(ctx context.Context, client *http.Client, method, url string, headers http.Header, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultBrowserUserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	return data, nil
}
