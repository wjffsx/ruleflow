// Package condition provides VPP condition nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package condition

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  SOCMonitorCondition — SOC 监控条件
// ─────────────────────────────────────────────

// SOCMonitorCondition 电池荷电状态监控条件
type SOCMonitorCondition struct {
	IDValue            string   `json:"id"`
	ChargeHighLimit    *float64 `json:"charge_high_limit"`    // 充电态 SOC 上限（%）
	ChargeLowLimit     *float64 `json:"charge_low_limit"`     // 充电态 SOC 下限（%）
	DischargeHighLimit *float64 `json:"discharge_high_limit"` // 放电态 SOC 上限（%）
	DischargeLowLimit  *float64 `json:"discharge_low_limit"`  // 放电态 SOC 下限（%）
	ChargeStatePoint   string   `json:"charge_state_point"`   // 充放电状态数据点名

	// 预编译默认值
	chHigh float64
	chLow  float64
	dsHigh float64
	dsLow  float64
}

// NewSOCMonitorCondition 创建 SOC 监控条件
func NewSOCMonitorCondition(id string, chargeHigh, chargeLow, dischargeHigh, dischargeLow *float64, chargeStatePoint string) *SOCMonitorCondition {
	c := &SOCMonitorCondition{
		IDValue:          id,
		ChargeStatePoint: chargeStatePoint,
	}
	if chargeHigh != nil {
		c.chHigh = *chargeHigh
	}
	if chargeLow != nil {
		c.chLow = *chargeLow
	}
	if dischargeHigh != nil {
		c.dsHigh = *dischargeHigh
	}
	if dischargeLow != nil {
		c.dsLow = *dischargeLow
	}
	return c
}

func (c *SOCMonitorCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	soc := data.Value()

	// 判断充放电状态
	charging := false
	if mdc, ok := data.(types.MultiDataContextInterface); ok && c.ChargeStatePoint != "" {
		val, err := mdc.GetPoint(c.ChargeStatePoint)
		if err == nil && val > 0.5 {
			charging = true
		}
	}

	if charging {
		if c.chHigh > 0 && soc >= c.chHigh {
			return true
		}
		if c.chLow > 0 && soc <= c.chLow {
			return true
		}
	} else {
		if c.dsHigh > 0 && soc >= c.dsHigh {
			return true
		}
		if c.dsLow > 0 && soc <= c.dsLow {
			return true
		}
	}
	return false
}

func (c *SOCMonitorCondition) ID() string   { return c.IDValue }
func (c *SOCMonitorCondition) Type() string { return "soc_monitor" }
func (c *SOCMonitorCondition) Description() string {
	return "SOC monitor with charge/discharge state correlation"
}