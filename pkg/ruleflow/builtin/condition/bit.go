// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  BitMaskCondition — 位掩码匹配条件
// ─────────────────────────────────────────────

// BitMaskCondition 位掩码匹配条件
// 用于检测数据值的特定位是否匹配期望值
type BitMaskCondition struct {
	IDValue  string `json:"id"`
	Mask     uint64 `json:"mask"`     // 位掩码
	Expected uint64 `json:"expected"` // 期望值
	Operator string `json:"operator"` // "and" | "or" | "eq"
}

// NewBitMaskCondition 创建位掩码匹配条件
func NewBitMaskCondition(id string, mask, expected uint64, operator string) *BitMaskCondition {
	if operator == "" {
		operator = "and"
	}
	return &BitMaskCondition{
		IDValue:  id,
		Mask:     mask,
		Expected: expected,
		Operator: operator,
	}
}

func (c *BitMaskCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	val := uint64(data.Value())
	switch c.Operator {
	case "and":
		return (val & c.Mask) == c.Expected
	case "or":
		return (val | c.Mask) == c.Expected
	case "eq":
		return val == c.Expected
	default:
		return false
	}
}

func (c *BitMaskCondition) ID() string   { return c.IDValue }
func (c *BitMaskCondition) Type() string { return "bit_mask" }
func (c *BitMaskCondition) Description() string {
	return fmt.Sprintf("bit_mask %s mask=%x expected=%x", c.Operator, c.Mask, c.Expected)
}
