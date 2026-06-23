package ext

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  ThresholdDetectCondition — 阈值检测条件
// ─────────────────────────────────────────────

// ThresholdDetectCondition 阈值检测条件节点。
// 支持多种比较运算符和可选持续时间（需配合 StateStore）。
//
// 运算符说明：
//   - gt: 大于 (>)
//   - lt: 小于 (<)
//   - gte: 大于等于 (>=)
//   - lte: 小于等于 (<=)
//   - eq: 等于 (==)
//   - neq: 不等于 (!=)
//   - outside_range: 超出范围 [min, max]
//   - inside_range: 在范围内 [min, max]
//
// 配置示例：
//
//	condition:
//	  leaf_type: "threshold_detect"
//	  leaf_config:
//	    operator: "gt"
//	    value: 253
//	    duration: "30s"       # 可选：持续满足条件 30s 后才触发
type ThresholdDetectCondition struct {
	IDValue  string
	Operator string        // gt / lt / gte / lte / eq / neq / outside_range / inside_range
	Value    float64       // 阈值（outside_range/inside_range 时作为 max）
	MinValue float64       // 仅 outside_range/inside_range 时作为 min
	Duration time.Duration // 持续时间（0 表示立即触发）
	store    core.StateStore
}

var _ core.Condition = (*ThresholdDetectCondition)(nil)

func NewThresholdDetectCondition(id, operator string, value float64, duration time.Duration, store core.StateStore) *ThresholdDetectCondition {
	return &ThresholdDetectCondition{
		IDValue:  id,
		Operator: operator,
		Value:    value,
		Duration: duration,
		store:    store,
	}
}

func (c *ThresholdDetectCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	val := data.Value()

	matched := false
	switch c.Operator {
	case "gt", "greater_than":
		matched = val > c.Value
	case "lt", "less_than":
		matched = val < c.Value
	case "gte", "greater_equal":
		matched = val >= c.Value
	case "lte", "less_equal":
		matched = val <= c.Value
	case "eq", "equal":
		matched = val == c.Value
	case "neq", "not_equal":
		matched = val != c.Value
	case "outside_range":
		matched = val < c.MinValue || val > c.Value
	case "inside_range":
		matched = val >= c.MinValue && val <= c.Value
	case "outside_limits":
		// 动态使用数据点自身上下限
		hi, hiOk := data.UpperLimit()
		lo, loOk := data.LowerLimit()
		if hiOk && loOk {
			matched = val > hi || val < lo
		} else if hiOk {
			matched = val > hi
		} else if loOk {
			matched = val < lo
		}
	default:
		matched = false
	}

	if !matched || c.Duration == 0 {
		return matched
	}

	// 持续时间检查：使用 StateStore 记录首次匹配时间
	if c.store == nil {
		return matched // 没有 StateStore 时无法做持续时间检查，直接返回匹配结果
	}

	fqn := data.FQN()
	key := fmt.Sprintf("threshold_dur:%s:%s", c.IDValue, fqn)
	now := time.Now()

	if val, ok := c.store.Get(key); ok {
		if startTime, ok := val.(time.Time); ok {
			if now.Sub(startTime) >= c.Duration {
				return true
			}
			return false
		}
	}

	// 首次匹配，记录时间
	c.store.Set(key, now)
	return false
}

func (c *ThresholdDetectCondition) ID() string        { return c.IDValue }
func (c *ThresholdDetectCondition) Type() string      { return "threshold_detect" }
func (c *ThresholdDetectCondition) Description() string {
	if c.Duration > 0 {
		return fmt.Sprintf("threshold %s %.2f for %v", c.Operator, c.Value, c.Duration)
	}
	return fmt.Sprintf("threshold %s %.2f", c.Operator, c.Value)
}
