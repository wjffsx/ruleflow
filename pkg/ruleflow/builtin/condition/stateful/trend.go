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
//  TrendCondition — 趋势判断条件
// ─────────────────────────────────────────────

// TrendCondition 判断值在时间窗口内的变化趋势
type TrendCondition struct {
	IDValue   string
	Direction string        // "increasing" / "decreasing"
	Window    time.Duration // 滑动窗口时长
	Threshold float64       // 变化百分比阈值 (0.1 = 10%)
}

func NewTrendCondition(id, direction string, window time.Duration, threshold float64) *TrendCondition {
	return &TrendCondition{
		IDValue:   id,
		Direction: direction,
		Window:    window,
		Threshold: threshold,
	}
}

// trendSample 趋势采样点
type trendSample struct {
	Value     float64
	Timestamp time.Time
}

// trendState 趋势条件状态
type trendState struct {
	Samples []trendSample
}

func (c *TrendCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false
	}

	key := util.StateKey("trend", c.IDValue, data.DeviceID(), data.PointName())
	now := time.Unix(0, data.Timestamp())
	if data.Timestamp() > 1e18 {
		now = time.UnixMilli(data.Timestamp())
	}

	// 加载或初始化状态
	stateI, loaded := sd.StateStore().Get(key)
	var state *trendState
	if loaded {
		if s, ok := stateI.(*trendState); ok {
			state = s
		}
	}
	if state == nil {
		state = &trendState{}
	}

	// 追加采样点
	state.Samples = append(state.Samples, trendSample{Value: data.Value(), Timestamp: now})

	// 淘汰过期采样点
	cutoff := now.Add(-c.Window)
	i := 0
	for i < len(state.Samples) && state.Samples[i].Timestamp.Before(cutoff) {
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

	// 计算变化率
	oldest := state.Samples[0]
	newest := state.Samples[len(state.Samples)-1]
	if oldest.Value == 0 {
		return false
	}
	changeRate := (newest.Value - oldest.Value) / abs64(oldest.Value)

	switch c.Direction {
	case "increasing":
		return changeRate >= c.Threshold
	case "decreasing":
		return changeRate <= -c.Threshold
	default:
		return false
	}
}

func (c *TrendCondition) ID() string   { return c.IDValue }
func (c *TrendCondition) Type() string { return "trend" }
func (c *TrendCondition) Description() string {
	return fmt.Sprintf("trend %s threshold %.2f", c.Direction, c.Threshold)
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
