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
//  DurationCondition — 持续时间条件
// ─────────────────────────────────────────────

// DurationCondition 内嵌条件连续满足指定时长后才返回 true
type DurationCondition struct {
	IDValue  string
	Inner    core.Condition // 内嵌条件
	Duration time.Duration  // 持续时间要求
}

func NewDurationCondition(id string, inner core.Condition, duration time.Duration) *DurationCondition {
	return &DurationCondition{IDValue: id, Inner: inner, Duration: duration}
}

func (c *DurationCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	// 检查是否有 StateStore
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false // 无状态存储，无法判断持续时间
	}

	key := util.StateKey("duration", c.IDValue, data.DeviceID(), data.PointName())
	innerResult := c.Inner.Evaluate(ctx, data)

	now := time.Unix(0, data.Timestamp())
	if data.Timestamp() > 1e18 {
		now = time.UnixMilli(data.Timestamp())
	}

	if innerResult {
		stateI, loaded := sd.StateStore().Get(key)
		if !loaded {
			sd.StateStore().Set(key, &now)
			return false
		}
		since, ok := stateI.(*time.Time)
		if !ok {
			sd.StateStore().Set(key, &now)
			return false
		}
		return now.Sub(*since) >= c.Duration
	}

	// 条件不满足，重置状态
	sd.StateStore().Delete(key)
	return false
}

func (c *DurationCondition) ID() string   { return c.IDValue }
func (c *DurationCondition) Type() string { return "duration" }
func (c *DurationCondition) Description() string {
	return fmt.Sprintf("duration %s", c.Duration)
}
