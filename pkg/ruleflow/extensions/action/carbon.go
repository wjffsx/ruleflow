// Package action provides VPP action nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package action

import (
	"context"
	"fmt"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  CarbonCalcAction — 碳排放计算动作
// ─────────────────────────────────────────────

// 缺省排放因子（tCO₂/MWh）
var defaultEmissionFactors = map[string]float64{
	"scope2": 0.5813, // 全国电网平均
	"diesel": 0.7400,
	"gas":    0.4100,
	"coal":   0.9500,
}

// CarbonCalcAction 碳排放计算动作
type CarbonCalcAction struct {
	IDValue        string  `json:"id"`
	Scope          string  `json:"scope"`           // scope1 / scope2
	EmissionFactor float64 `json:"emission_factor"` // tCO₂/MWh
	FuelType       string  `json:"fuel_type"`       // diesel / gas / coal
	PowerPoint     string  `json:"power_point"`     // 功率/用量数据点名
	OutputPoint    string  `json:"output_point"`    // 碳排放量结果输出点名
}

// NewCarbonCalcAction 创建碳排放计算动作
func NewCarbonCalcAction(id, scope string, emissionFactor float64, fuelType, powerPoint, outputPoint string) *CarbonCalcAction {
	return &CarbonCalcAction{
		IDValue:        id,
		Scope:          scope,
		EmissionFactor: emissionFactor,
		FuelType:       fuelType,
		PowerPoint:     powerPoint,
		OutputPoint:    outputPoint,
	}
}

func (a *CarbonCalcAction) Execute(ctx context.Context, data core.DataContext) error {
	// 1. 确定排放因子
	factor := a.EmissionFactor
	if factor == 0 {
		if a.Scope == "scope2" {
			factor = defaultEmissionFactors["scope2"]
		} else if f, ok := defaultEmissionFactors[a.FuelType]; ok {
			factor = f
		} else {
			return fmt.Errorf("carbon_calc: no emission factor for scope=%s fuel=%s", a.Scope, a.FuelType)
		}
	}

	// 2. 获取用量值
	var usage float64
	if a.PowerPoint != "" {
		mdc, ok := data.(types.MultiDataContextInterface)
		if !ok {
			return fmt.Errorf("carbon_calc requires MultiDataContext for point query")
		}
		v, err := mdc.GetPoint(a.PowerPoint)
		if err != nil {
			return fmt.Errorf("carbon_calc: get point %s: %w", a.PowerPoint, err)
		}
		usage = v
	} else {
		usage = data.Value()
	}

	// 3. 计算碳排放
	emission := usage * factor

	// 4. 写入结果
	data.SetTag("carbon_emission", fmt.Sprintf("%.4f", emission))
	data.SetTag("carbon_scope", a.Scope)
	data.SetTag("carbon_factor", fmt.Sprintf("%.4f", factor))
	if a.OutputPoint != "" {
		data.SetTag("_output_point", a.OutputPoint)
	}
	return nil
}

func (a *CarbonCalcAction) ID() string   { return a.IDValue }
func (a *CarbonCalcAction) Type() string { return "carbon_calc" }
func (a *CarbonCalcAction) Description() string {
	return fmt.Sprintf("carbon calc scope=%s factor=%.4f", a.Scope, a.EmissionFactor)
}