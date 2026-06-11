// Package stateful provides stateful condition nodes
package stateful

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  StateChangeCondition — 状态变化条件
// ─────────────────────────────────────────────

// StateChangeCondition 检测数据点值的变化
type StateChangeCondition struct {
	IDValue string
	From    *float64 // 旧值（nil=任意变化）
	To      *float64 // 新值（nil=任意变化）
}

func NewStateChangeCondition(id string, from, to *float64) *StateChangeCondition {
	return &StateChangeCondition{IDValue: id, From: from, To: to}
}

func (c *StateChangeCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	prev, ok := data.PreviousValue()
	if !ok {
		return false // 首次评估，无前值
	}
	curr := data.Value()

	// from 检查
	if c.From != nil && prev != *c.From {
		return false
	}
	// to 检查
	if c.To != nil && curr != *c.To {
		return false
	}
	// 值必须发生变化
	return prev != curr
}

func (c *StateChangeCondition) ID() string   { return c.IDValue }
func (c *StateChangeCondition) Type() string { return "state_change" }
func (c *StateChangeCondition) Description() string {
	return fmt.Sprintf("state change from %v to %v", c.From, c.To)
}
