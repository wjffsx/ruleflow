// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  RateLimitCondition — 变化率越限检测条件（瞬时版本）
// ─────────────────────────────────────────────

// RateLimitCondition 变化率越限检测条件（瞬时版本）
// 检测数据值的变化率是否超过阈值
// 无状态实现，使用 PreviousValue() + Timestamp() 计算
type RateLimitCondition struct {
	IDValue       string  `json:"id"`
	RateThreshold float64 `json:"rate_threshold"` // 变化率阈值 (单位/秒)
	Direction     string  `json:"direction"`      // "up" | "down" | "both"
}

// NewRateLimitCondition 创建变化率越限检测条件（瞬时版本）
func NewRateLimitCondition(id string, rateThreshold float64, direction string) *RateLimitCondition {
	if direction == "" {
		direction = "both"
	}
	return &RateLimitCondition{
		IDValue:       id,
		RateThreshold: rateThreshold,
		Direction:     direction,
	}
}

func (c *RateLimitCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	prev, ok := data.PreviousValue()
	if !ok {
		return false // 首次评估，无前值
	}

	// 前值时间戳由上游设置到 Tag（或扩展 DataContext 提供 PreviousTimestamp()）
	prevTsStr := data.GetTag("_prev_timestamp")
	if prevTsStr == "" {
		return false // 无前值时间戳
	}

	prevTs, err := strconv.ParseInt(prevTsStr, 10, 64)
	if err != nil {
		return false // 时间戳解析失败
	}

	currentTs := data.Timestamp()
	elapsed := float64(currentTs - prevTs) / 1000.0 // 毫秒转秒

	if elapsed <= 0 {
		return false // 时间戳异常
	}

	rate := (data.Value() - prev) / elapsed

	switch c.Direction {
	case "up":
		return rate >= c.RateThreshold
	case "down":
		return rate <= -c.RateThreshold
	case "both":
		return math.Abs(rate) >= c.RateThreshold
	default:
		return false
	}
}

func (c *RateLimitCondition) ID() string   { return c.IDValue }
func (c *RateLimitCondition) Type() string { return "rate_limit" }
func (c *RateLimitCondition) Description() string {
	return fmt.Sprintf("rate limit %s %.2f/s", c.Direction, c.RateThreshold)
}