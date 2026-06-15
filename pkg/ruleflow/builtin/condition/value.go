// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"
	"sort"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  值相关条件
// ─────────────────────────────────────────────

// ValueRangeCondition 值范围条件
type ValueRangeCondition struct {
	IDValue  string   `json:"id"`
	MinValue *float64 `json:"min_value"`
	MaxValue *float64 `json:"max_value"`
}

func NewValueRangeCondition(id string, min, max *float64) *ValueRangeCondition {
	return &ValueRangeCondition{IDValue: id, MinValue: min, MaxValue: max}
}

func (c *ValueRangeCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	v := data.Value()
	if c.MinValue != nil && v < *c.MinValue {
		return false
	}
	if c.MaxValue != nil && v > *c.MaxValue {
		return false
	}
	return true
}

func (c *ValueRangeCondition) ID() string   { return c.IDValue }
func (c *ValueRangeCondition) Type() string { return "value_range" }
func (c *ValueRangeCondition) Description() string {
	return fmt.Sprintf("value in [%v, %v]", c.MinValue, c.MaxValue)
}

// ValueInCondition 离散值匹配条件
// 判断值是否在指定的离散值列表中
// 使用预排序 + 二分查找，避免 float64 作为 map key 的 NaN/精度问题
type ValueInCondition struct {
	IDValue      string
	Values       []float64
	sortedValues []float64 // 预排序，用于二分查找
}

func NewValueInCondition(id string, values []float64) *ValueInCondition {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	return &ValueInCondition{IDValue: id, Values: values, sortedValues: sorted}
}

func (c *ValueInCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	v := data.Value()
	idx := sort.SearchFloat64s(c.sortedValues, v)
	return idx < len(c.sortedValues) && c.sortedValues[idx] == v
}

func (c *ValueInCondition) ID() string          { return c.IDValue }
func (c *ValueInCondition) Type() string        { return "value_in" }
func (c *ValueInCondition) Description() string { return fmt.Sprintf("value in %v", c.Values) }
