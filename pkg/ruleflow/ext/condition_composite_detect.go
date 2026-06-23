package ext

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  CompositeDetectCondition — 复合条件检测节点
// ─────────────────────────────────────────────

// CompositeDetectCondition 复合条件检测节点。
// 支持 AND/OR 逻辑组合多个子条件，用于实现跨字段的复合告警规则。
//
// 配置示例：
//
//	condition:
//	  leaf_type: "composite_detect"
//	  leaf_config:
//	    logic: "and"                 # and / or
//	    conditions:
//	      - operator: "gt"
//	        field: "value"
//	        threshold: 253
//	      - operator: "lt"
//	        field: "value"
//	        threshold: 50.5
type CompositeDetectCondition struct {
	IDValue    string
	Logic      string             // "and" / "or"
	Conditions []SubCondition     // 子条件列表
}

// SubCondition 子条件
type SubCondition struct {
	Operator  string  `json:"operator"`
	Field     string  `json:"field"`
	Threshold float64 `json:"threshold"`
	MinValue  float64 `json:"min_value,omitempty"` // 用于 outside_range/inside_range
}

var _ core.Condition = (*CompositeDetectCondition)(nil)

func NewCompositeDetectCondition(id, logic string, conditions []SubCondition) *CompositeDetectCondition {
	if logic == "" {
		logic = "and"
	}
	return &CompositeDetectCondition{
		IDValue:    id,
		Logic:      logic,
		Conditions: conditions,
	}
}

func (c *CompositeDetectCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	if len(c.Conditions) == 0 {
		return false
	}

	val := data.Value()

	for i, cond := range c.Conditions {
		matched := evaluateSingle(cond, val)

		if c.Logic == "and" && !matched {
			return false
		}
		if c.Logic == "or" && matched {
			return true
		}
		_ = i
	}

	// AND: 全部匹配则 true；OR: 全部不匹配则 false
	if c.Logic == "and" {
		return true
	}
	return false // or 逻辑下所有条件都不匹配
}

func evaluateSingle(cond SubCondition, val float64) bool {
	switch cond.Operator {
	case "gt", "greater_than":
		return val > cond.Threshold
	case "lt", "less_than":
		return val < cond.Threshold
	case "gte", "greater_equal":
		return val >= cond.Threshold
	case "lte", "less_equal":
		return val <= cond.Threshold
	case "eq", "equal":
		return val == cond.Threshold
	case "neq", "not_equal":
		return val != cond.Threshold
	case "outside_range":
		return val < cond.MinValue || val > cond.Threshold
	case "inside_range":
		return val >= cond.MinValue && val <= cond.Threshold
	default:
		return false
	}
}

func (c *CompositeDetectCondition) ID() string        { return c.IDValue }
func (c *CompositeDetectCondition) Type() string      { return "composite_detect" }
func (c *CompositeDetectCondition) Description() string {
	return fmt.Sprintf("composite detect logic=%s conditions=%d", c.Logic, len(c.Conditions))
}
