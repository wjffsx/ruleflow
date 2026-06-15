// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"
	"math"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DeltaThresholdCondition — 增量阈值检测条件
// ─────────────────────────────────────────────

// DeltaThresholdCondition 增量阈值检测条件
// 检测数据值的变化量是否超过阈值
// 无状态实现，使用 DataContext.PreviousValue() 获取前值
type DeltaThresholdCondition struct {
	IDValue   string  `json:"id"`
	Threshold float64 `json:"threshold"` // 变化量阈值
	Direction string  `json:"direction"` // "up" | "down" | "both"
}

// NewDeltaThresholdCondition 创建增量阈值检测条件
func NewDeltaThresholdCondition(id string, threshold float64, direction string) *DeltaThresholdCondition {
	if direction == "" {
		direction = "both"
	}
	return &DeltaThresholdCondition{
		IDValue:   id,
		Threshold: threshold,
		Direction: direction,
	}
}

func (c *DeltaThresholdCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	prev, ok := data.PreviousValue()
	if !ok {
		return false // 首次评估，无前值
	}

	delta := data.Value() - prev

	switch c.Direction {
	case "up":
		return delta >= c.Threshold
	case "down":
		return delta <= -c.Threshold
	case "both":
		return math.Abs(delta) >= c.Threshold
	default:
		return false
	}
}

func (c *DeltaThresholdCondition) ID() string   { return c.IDValue }
func (c *DeltaThresholdCondition) Type() string { return "delta_threshold" }
func (c *DeltaThresholdCondition) Description() string {
	return fmt.Sprintf("delta threshold %s %.2f", c.Direction, c.Threshold)
}
