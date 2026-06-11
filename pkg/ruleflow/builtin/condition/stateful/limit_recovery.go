// Package stateful provides stateful condition nodes
package stateful

import (
	"context"
	"fmt"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes/util"
)

// ─────────────────────────────────────────────
//  LimitRecoveryCondition — 越限恢复检测条件
// ─────────────────────────────────────────────

// LimitRecoveryState 越限恢复状态
type LimitRecoveryState struct {
	WasExceeded      bool  // 之前是否越限
	RecoveryStartTime int64 // 恢复开始时间（毫秒）
}

// LimitRecoveryCondition 越限恢复检测条件
// 检测值是否从越限状态恢复到正常范围，并持续指定时间
type LimitRecoveryCondition struct {
	IDValue    string        `json:"id"`
	Duration   time.Duration `json:"duration"`   // 恢复后持续多久才触发
	Hysteresis float64       `json:"hysteresis"` // 迟滞值，防止临界值抖动
}

// NewLimitRecoveryCondition 创建越限恢复检测条件
func NewLimitRecoveryCondition(id string, duration time.Duration, hysteresis float64) *LimitRecoveryCondition {
	return &LimitRecoveryCondition{
		IDValue:    id,
		Duration:   duration,
		Hysteresis: hysteresis,
	}
}

func (c *LimitRecoveryCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false // 无 StateStore 支持
	}

	key := util.StateKey("limit_recovery", c.IDValue, data.DeviceID(), data.PointName())
	nowTs := data.Timestamp()
	val := data.Value()

	// 获取上下限
	upper, hasUpper := data.UpperLimit()
	lower, hasLower := data.LowerLimit()

	// 判断当前是否越限（考虑迟滞）
	isExceeded := false
	if hasUpper && val > upper {
		isExceeded = true
	}
	if hasLower && val < lower {
		isExceeded = true
	}

	// 判断是否恢复（考虑迟滞）
	isRecovered := true
	if hasUpper && val >= upper - c.Hysteresis {
		isRecovered = false
	}
	if hasLower && val <= lower + c.Hysteresis {
		isRecovered = false
	}

	// 加载或初始化状态
	stateI, loaded := sd.StateStore().Get(key)
	var state *LimitRecoveryState
	if loaded {
		if s, ok := stateI.(*LimitRecoveryState); ok {
			state = s
		}
	}
	if state == nil {
		state = &LimitRecoveryState{}
	}

	// 更新状态
	if isExceeded {
		// 当前越限，标记状态
		state.WasExceeded = true
		state.RecoveryStartTime = 0
		sd.StateStore().Set(key, state)
		return false
	}

	if state.WasExceeded && isRecovered {
		// 从越限恢复
		if state.RecoveryStartTime == 0 {
			// 首次恢复，记录时间
			state.RecoveryStartTime = nowTs
			sd.StateStore().Set(key, state)
			return false
		}

		// 检查是否持续足够时间
		elapsed := nowTs - state.RecoveryStartTime
		if elapsed >= c.Duration.Milliseconds() {
			// 恢复持续足够时间，触发条件
			// 重置状态（可选：保持状态用于后续检测）
			sd.StateStore().Set(key, state)
			return true
		}

		sd.StateStore().Set(key, state)
		return false
	}

	// 正常状态，无变化
	sd.StateStore().Set(key, state)
	return false
}

func (c *LimitRecoveryCondition) ID() string   { return c.IDValue }
func (c *LimitRecoveryCondition) Type() string { return "limit_recovery" }
func (c *LimitRecoveryCondition) Description() string {
	return fmt.Sprintf("limit recovery after %v with hysteresis %.2f", c.Duration, c.Hysteresis)
}

// abs helper function
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}