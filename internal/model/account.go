package model

import "time"

type Status string

const (
	StatusChecking    Status = "checking"
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
	StatusPending     Status = "pending"
	StatusUnknown     Status = "unknown"
)

type Account struct {
	ID            string `json:"id"`
	Channel       string `json:"channel"`
	ChannelName   string `json:"channel_name"`
	Country       string `json:"country"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Status        Status `json:"status"`
	StatusMessage string `json:"status_message"`
	StatusLabel   string `json:"status_label"`
	RawStatus     int    `json:"raw_status,omitempty"`
	Priority      int    `json:"priority,omitempty"`
	UpdatedAt     string `json:"updated_at"`
	SourceURL     string `json:"source_url,omitempty"`
}

type ChannelState struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Order        int    `json:"order"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	AccountCount int    `json:"account_count"`
	UpdatedAt    string `json:"updated_at"`
}

type StatusInfo struct {
	Status      Status `json:"status"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type Warning struct {
	Level   string `json:"level"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Snapshot struct {
	Code             int            `json:"code"`
	Message          string         `json:"message"`
	GeneratedAt      time.Time      `json:"generated_at"`
	CacheTTLSeconds  int            `json:"cache_ttl_seconds"`
	Accounts         []Account      `json:"accounts"`
	Channels         []ChannelState `json:"channels"`
	Warnings         []Warning      `json:"warnings"`
	StatusLegend     []StatusInfo   `json:"status_legend"`
	AvailableCount   int            `json:"available_count"`
	UnavailableCount int            `json:"unavailable_count"`
	PendingCount     int            `json:"pending_count"`
	TotalCount       int            `json:"total_count"`
}
