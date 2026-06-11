// Package contract - 限流契约
package contract

import "context"

// Limiter 限流器接口。
type Limiter interface {
	Allow(key string) bool
	Wait(ctx context.Context, key string) error
}

// NoopLimiter 返回不限流的 Limiter。
func NoopLimiter() Limiter { return noopLimiter{} }

type noopLimiter struct{}

func (noopLimiter) Allow(_ string) bool                    { return true }
func (noopLimiter) Wait(_ context.Context, _ string) error { return nil }

// LimiterKey 构造引擎内部限流 key。
func LimiterKey(chainID string) string {
	return "chain:" + chainID
}

// LimiterKeyForRule 构造规则级限流 key
func LimiterKeyForRule(chainID, ruleID string) string {
	return "chain:" + chainID + ":rule:" + ruleID
}
