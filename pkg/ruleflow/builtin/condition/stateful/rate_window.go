// Package stateful provides stateful condition nodes
package stateful

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes/util"
)

// ─────────────────────────────────────────────
//  RateLimitWindowCondition — 变化率越限检测条件（窗口版本）
// ─────────────────────────────────────────────

// rateSample 变化率采样点
type rateSample struct {
	Value     float64
	Timestamp int64
}

// RateLimitWindowState 变化率窗口状态
type RateLimitWindowState struct {
	Samples []rateSample
}

// RateLimitWindowCondition 变化率越限检测条件（窗口版本）
// 使用 StateStore 维护滑动窗口，计算窗口内平均变化率
type RateLimitWindowCondition struct {
	IDValue       string        `json:"id"`
	RateThreshold float64       `json:"rate_threshold"` // 变化率阈值 (单位/秒)
	Window        time.Duration `json:"window"`         // 计算窗口
}

// NewRateLimitWindowCondition 创建变化率越限检测条件（窗口版本）
func NewRateLimitWindowCondition(id string, rateThreshold float64, window time.Duration) *RateLimitWindowCondition {
	return &RateLimitWindowCondition{
		IDValue:       id,
		RateThreshold: rateThreshold,
		Window:        window,
	}
}

func (c *RateLimitWindowCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false // 无 StateStore 支持
	}

	key := util.StateKey("rate_limit", c.IDValue, data.DeviceID(), data.PointName())
	nowTs := data.Timestamp()

	// 加载或初始化状态
	stateI, loaded := sd.StateStore().Get(key)
	var state *RateLimitWindowState
	if loaded {
		if s, ok := stateI.(*RateLimitWindowState); ok {
			state = s
		}
	}
	if state == nil {
		state = &RateLimitWindowState{}
	}

	// 追加采样点
	state.Samples = append(state.Samples, rateSample{
		Value:     data.Value(),
		Timestamp: nowTs,
	})

	// 淘汰过期采样点
	cutoff := nowTs - c.Window.Milliseconds()
	i := 0
	for i < len(state.Samples) && state.Samples[i].Timestamp < cutoff {
		i++
	}
	if i > 0 {
		state.Samples = state.Samples[i:]
	}

	sd.StateStore().Set(key, state)

	// 至少需要 2 个采样点
	if len(state.Samples) < 2 {
		return false
	}

	// 计算窗口内平均变化率
	oldest := state.Samples[0]
	newest := state.Samples[len(state.Samples)-1]
	elapsed := float64(newest.Timestamp - oldest.Timestamp) / 1000.0

	if elapsed <= 0 {
		return false
	}

	avgRate := math.Abs(newest.Value - oldest.Value) / elapsed
	return avgRate > c.RateThreshold
}

func (c *RateLimitWindowCondition) ID() string   { return c.IDValue }
func (c *RateLimitWindowCondition) Type() string { return "rate_limit_window" }
func (c *RateLimitWindowCondition) Description() string {
	return fmt.Sprintf("rate limit window %.2f/s over %v", c.RateThreshold, c.Window)
}