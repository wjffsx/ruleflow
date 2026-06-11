// Package stateful provides stateful condition nodes
package stateful

import (
	"context"
	"fmt"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DynamicThresholdCondition — 动态阈值条件
// ─────────────────────────────────────────────

// DynamicThresholdCondition 从 DataContext 的 Tag 或 Raw() 中读取阈值
type DynamicThresholdCondition struct {
	IDValue  string
	Operator string // "gt" / "lt" / "gte" / "lte" / "eq" / "neq"
	Source   string // "tag:key_name" 或 "raw:field.path"
}

func NewDynamicThresholdCondition(id, operator, source string) *DynamicThresholdCondition {
	return &DynamicThresholdCondition{IDValue: id, Operator: operator, Source: source}
}

func (c *DynamicThresholdCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	threshold := c.resolveThreshold(data)
	if threshold == nil {
		return false
	}

	val := data.Value()
	switch c.Operator {
	case "gt":
		return val > *threshold
	case "lt":
		return val < *threshold
	case "gte":
		return val >= *threshold
	case "lte":
		return val <= *threshold
	case "eq":
		return val == *threshold
	case "neq":
		return val != *threshold
	default:
		return false
	}
}

func (c *DynamicThresholdCondition) resolveThreshold(data core.DataContext) *float64 {
	if len(c.Source) > 4 && c.Source[:4] == "tag:" {
		tagKey := c.Source[4:]
		tagVal := data.GetTag(tagKey)
		if tagVal == "" {
			return nil
		}
		var v float64
		if _, err := fmt.Sscanf(tagVal, "%f", &v); err == nil {
			return &v
		}
	}
	return nil
}

func (c *DynamicThresholdCondition) ID() string   { return c.IDValue }
func (c *DynamicThresholdCondition) Type() string { return "dynamic_threshold" }
func (c *DynamicThresholdCondition) Description() string {
	return fmt.Sprintf("dynamic threshold %s %s", c.Operator, c.Source)
}
