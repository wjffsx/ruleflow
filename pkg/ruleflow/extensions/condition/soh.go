// Package condition provides VPP condition nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package condition

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  SOHEstimatorCondition — 电池健康状态估算条件
// ─────────────────────────────────────────────

// SOHEstimatorCondition 电池健康状态估算条件
type SOHEstimatorCondition struct {
	IDValue                 string  `json:"id"`
	SOHThreshold            float64 `json:"soh_threshold"`
	CycleCountPoint         string  `json:"cycle_count_point"`
	InternalResistancePoint string  `json:"internal_resistance_point"`
	CapacityPoint           string  `json:"capacity_point"`
	NominalCapacity         float64 `json:"nominal_capacity"`
	Method                  string  `json:"method"`
}

// NewSOHEstimatorCondition 创建电池健康状态估算条件
func NewSOHEstimatorCondition(id string, sohThreshold float64, cycleCountPoint, irPoint, capacityPoint string, nominalCapacity float64, method string) *SOHEstimatorCondition {
	if sohThreshold == 0 {
		sohThreshold = 80
	}
	if method == "" {
		method = "capacity_ratio"
	}
	return &SOHEstimatorCondition{
		IDValue:                 id,
		SOHThreshold:            sohThreshold,
		CycleCountPoint:         cycleCountPoint,
		InternalResistancePoint: irPoint,
		CapacityPoint:           capacityPoint,
		NominalCapacity:         nominalCapacity,
		Method:                  method,
	}
}

func (c *SOHEstimatorCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	mdc, ok := data.(types.MultiDataContextInterface)
	if !ok {
		return false
	}

	switch c.Method {
	case "capacity_ratio":
		currentCap, err := mdc.GetPoint(c.CapacityPoint)
		if err != nil || c.NominalCapacity <= 0 {
			return false
		}
		soh := currentCap / c.NominalCapacity * 100
		return soh < c.SOHThreshold

	case "resistance_based":
		ir, err := mdc.GetPoint(c.InternalResistancePoint)
		if err != nil {
			return false
		}
		soh := 100.0 - ir*40.0
		return soh < c.SOHThreshold

	default:
		return false
	}
}

func (c *SOHEstimatorCondition) ID() string   { return c.IDValue }
func (c *SOHEstimatorCondition) Type() string { return "soh_estimator" }
func (c *SOHEstimatorCondition) Description() string {
	return fmt.Sprintf("SOH estimator threshold=%.0f%% method=%s", c.SOHThreshold, c.Method)
}