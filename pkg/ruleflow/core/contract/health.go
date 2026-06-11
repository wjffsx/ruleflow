// Package contract - 健康检查契约
package contract

import (
	"context"
	"time"
)

// HealthStatus 引擎健康状态摘要（传输无关版本，仅包含引擎自身状态）
type HealthStatus struct {
	Status        string    `json:"status"`
	LoadedChains  int       `json:"loaded_chains"`
	ActiveEval    int64     `json:"active_eval"`
	ShuttingDown  bool      `json:"shutting_down"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	Uptime        string    `json:"uptime,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
}

// LivenessChecker 存活检查
type LivenessChecker interface {
	IsAlive(ctx context.Context) bool
}

// ReadinessChecker 就绪检查
type ReadinessChecker interface {
	IsReady(ctx context.Context) (ready bool, status HealthStatus)
}

// StatusReporter 详细状态报告
type StatusReporter interface {
	ReportStatus(ctx context.Context) HealthStatus
}

// ChainLister 链列表提供者
type ChainLister interface {
	ListChains(ctx context.Context) ([]string, error)
}

// HealthProvider 完整健康可观测性接口（组合接口，方便一键注入）
type HealthProvider interface {
	LivenessChecker
	ReadinessChecker
	StatusReporter
	ChainLister
}
