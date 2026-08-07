package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Q1YZn/appleshare-hub/internal/model"
	"github.com/Q1YZn/appleshare-hub/internal/provider"
)

type Service struct {
	providers []provider.Provider
	cacheTTL  time.Duration

	mu       sync.Mutex
	cached   model.Snapshot
	cachedAt time.Time
	inflight *call
}

type call struct {
	done chan struct{}
	snap model.Snapshot
}

func New(providers []provider.Provider, cacheTTL time.Duration) *Service {
	return &Service{
		providers: providers,
		cacheTTL:  cacheTTL,
	}
}

func (s *Service) Snapshot(ctx context.Context) (model.Snapshot, error) {
	s.mu.Lock()
	if !s.cachedAt.IsZero() && time.Since(s.cachedAt) < s.cacheTTL {
		snap := s.cached
		s.mu.Unlock()
		return snap, nil
	}
	if s.inflight != nil {
		wait := s.inflight
		s.mu.Unlock()
		<-wait.done
		return wait.snap, nil
	}

	current := &call{done: make(chan struct{})}
	s.inflight = current
	s.mu.Unlock()

	current.snap = s.build(ctx)

	s.mu.Lock()
	s.cached = current.snap
	s.cachedAt = time.Now()
	s.inflight = nil
	close(current.done)
	s.mu.Unlock()
	return current.snap, nil
}

func (s *Service) build(ctx context.Context) model.Snapshot {
	type result struct {
		channel  model.ChannelState
		accounts []model.Account
	}

	results := make(chan result, len(s.providers))
	var wg sync.WaitGroup
	for i, p := range s.providers {
		wg.Add(1)
		go func(i int, p provider.Provider) {
			defer wg.Done()
			accounts, err := p.Fetch(ctx)
			state := model.ChannelState{
				ID:        p.ID(),
				Name:      p.Name(),
				Order:     i,
				Status:    "ok",
				UpdatedAt: time.Now().Format(time.RFC3339),
			}
			if err != nil {
				state.Status = "error"
				state.Error = err.Error()
			}
			state.AccountCount = len(accounts)
			if state.Status == "ok" && len(accounts) == 0 {
				state.Status = "empty"
			}
			results <- result{channel: state, accounts: accounts}
		}(i, p)
	}
	wg.Wait()
	close(results)

	snap := model.Snapshot{
		Code:            200,
		Message:         "ok",
		GeneratedAt:     time.Now(),
		CacheTTLSeconds: int(s.cacheTTL.Seconds()),
		Warnings:        defaultWarnings(),
		StatusLegend:    statusLegend(),
	}

	for res := range results {
		snap.Channels = append(snap.Channels, res.channel)
		for _, account := range res.accounts {
			snap.Accounts = append(snap.Accounts, account)
		}
	}

	sort.Slice(snap.Accounts, func(i, j int) bool {
		if snap.Accounts[i].Status != snap.Accounts[j].Status {
			return accountStatusRank(snap.Accounts[i].Status) < accountStatusRank(snap.Accounts[j].Status)
		}
		if snap.Accounts[i].Priority != snap.Accounts[j].Priority {
			return snap.Accounts[i].Priority < snap.Accounts[j].Priority
		}
		return snap.Accounts[i].Channel < snap.Accounts[j].Channel
	})
	sort.Slice(snap.Channels, func(i, j int) bool {
		if snap.Channels[i].Order != snap.Channels[j].Order {
			return snap.Channels[i].Order < snap.Channels[j].Order
		}
		return snap.Channels[i].ID < snap.Channels[j].ID
	})

	for _, account := range snap.Accounts {
		snap.TotalCount++
		switch account.Status {
		case model.StatusAvailable:
			snap.AvailableCount++
		case model.StatusUnavailable:
			snap.UnavailableCount++
		default:
			snap.PendingCount++
		}
	}

	hasError := false
	for _, channel := range snap.Channels {
		if channel.Status == "error" {
			hasError = true
			break
		}
	}
	if hasError {
		snap.Message = "partial"
	}
	if len(snap.Channels) == 0 {
		snap.Message = "no_provider"
	}
	return snap
}

func accountStatusRank(status model.Status) int {
	switch status {
	case model.StatusAvailable:
		return 0
	case model.StatusChecking:
		return 1
	case model.StatusPending:
		return 2
	case model.StatusUnavailable:
		return 3
	default:
		return 4
	}
}

func defaultWarnings() []model.Warning {
	return []model.Warning{
		{
			Level:   "danger",
			Title:   "只允许在 App Store 登录",
			Content: "共享账号仅可用于 App Store 下载 App。不要使用这些账号登录设置、iCloud、App 与网站，也不要在网页端登录，否则可能触发设备锁机风险。",
		},
		{
			Level:   "danger",
			Title:   "iOS 26 退出入口在设置中",
			Content: "iOS 26 及之后版本，媒体与购买项目退出入口已移动到：设置 > 你的姓名/Apple 账户 > 媒体与购买项目 > 退出登录。使用后请及时退出。",
		},
		{
			Level:   "info",
			Title:   "账号状态以上游检测结果为准",
			Content: "页面状态来自上游渠道的检测结果，短时间缓存后仍可能变化。账号异常时请勿强制使用。",
		},
	}
}

func statusLegend() []model.StatusInfo {
	return []model.StatusInfo{
		{Status: model.StatusAvailable, Label: "可用", Description: "检测正常，可登录 App Store"},
		{Status: model.StatusChecking, Label: "检测中", Description: "账号正在检测，请稍后刷新"},
		{Status: model.StatusPending, Label: "待检测", Description: "账号等待检测"},
		{Status: model.StatusUnavailable, Label: "异常", Description: "账号异常，请勿使用"},
	}
}
