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
//  PeriodicCondition — 周期判断条件
// ─────────────────────────────────────────────

// PeriodicCondition 判断条件在连续 N 个周期内是否满足
type PeriodicCondition struct {
	IDValue string
	Inner   core.Condition // 内嵌条件
	Period  time.Duration  // 周期时长
	Count   int            // 连续满足次数
}

func NewPeriodicCondition(id string, inner core.Condition, period time.Duration, count int) *PeriodicCondition {
	return &PeriodicCondition{IDValue: id, Inner: inner, Period: period, Count: count}
}

// periodicState 周期条件状态
type periodicState struct {
	LastCheck   time.Time
	Consecutive int
}

func (c *PeriodicCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false
	}

	key := util.StateKey("periodic", c.IDValue, data.DeviceID(), data.PointName())
	now := time.Unix(0, data.Timestamp())
	if data.Timestamp() > 1e18 {
		now = time.UnixMilli(data.Timestamp())
	}

	innerResult := c.Inner.Evaluate(ctx, data)

	stateI, loaded := sd.StateStore().Get(key)
	var state *periodicState
	if loaded {
		if s, ok := stateI.(*periodicState); ok {
			state = s
		}
	}
	if state == nil {
		state = &periodicState{}
	}

	// 检查是否在新周期内
	if now.Sub(state.LastCheck) >= c.Period {
		state.LastCheck = now
		if innerResult {
			state.Consecutive++
		} else {
			state.Consecutive = 0
		}
	}

	sd.StateStore().Set(key, state)
	return state.Consecutive >= c.Count
}

func (c *PeriodicCondition) ID() string   { return c.IDValue }
func (c *PeriodicCondition) Type() string { return "periodic" }
func (c *PeriodicCondition) Description() string {
	return fmt.Sprintf("periodic %s x%d", c.Period, c.Count)
}
