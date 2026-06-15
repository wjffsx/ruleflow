// Package action provides VPP action nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  DeltaAccumulatorAction — 累积电量计算动作
// ─────────────────────────────────────────────

// accumulatorState 累积器状态
type accumulatorState struct {
	LastValue   float64
	Accumulated float64
	LastTime    time.Time
	Date        string
}

// DeltaAccumulatorAction 累积电量计算动作
type DeltaAccumulatorAction struct {
	IDValue         string `json:"id"`
	Key             string `json:"key"`
	Period          string `json:"period"`
	SourcePoint     string `json:"source_point"`
	OutputPoint     string `json:"output_point"`
	ResetAtMidnight bool   `json:"reset_at_midnight"`
}

// NewDeltaAccumulatorAction 创建累积电量计算动作
func NewDeltaAccumulatorAction(id, key, period, sourcePoint, outputPoint string, resetAtMidnight bool) *DeltaAccumulatorAction {
	if period == "" {
		period = "daily"
	}
	return &DeltaAccumulatorAction{
		IDValue:         id,
		Key:             key,
		Period:          period,
		SourcePoint:     sourcePoint,
		OutputPoint:     outputPoint,
		ResetAtMidnight: resetAtMidnight,
	}
}

func (a *DeltaAccumulatorAction) Execute(ctx context.Context, data core.DataContext) error {
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return fmt.Errorf("delta_accumulator requires StatefulDataContext")
	}

	// 当前值
	var currentVal float64
	if a.SourcePoint != "" {
		mdc, ok := data.(types.MultiDataContextInterface)
		if !ok {
			return fmt.Errorf("delta_accumulator: source_point requires MultiDataContext")
		}
		v, err := mdc.GetPoint(a.SourcePoint)
		if err != nil {
			return err
		}
		currentVal = v
	} else {
		currentVal = data.Value()
	}

	now := time.UnixMilli(data.Timestamp())
	stateKey := fmt.Sprintf("delta:%s:%s:%s", a.Key, a.Period, data.DeviceID())

	var state accumulatorState
	if raw, loaded := sd.StateStore().Get(stateKey); loaded {
		if s, ok := raw.(*accumulatorState); ok {
			state = *s
		}
	}

	// 周期重置
	currentDate := now.Format("2006-01-02")
	switch a.Period {
	case "daily":
		if state.Date != currentDate {
			state.Accumulated = 0
			state.Date = currentDate
		}
	case "monthly":
		currentMonth := now.Format("2006-01")
		if len(state.Date) >= 7 && state.Date[:7] != currentMonth {
			state.Accumulated = 0
			state.Date = currentMonth
		}
	}

	// 计算增量
	if !state.LastTime.IsZero() {
		delta := currentVal - state.LastValue
		if delta > 0 {
			state.Accumulated += delta
		}
	}
	state.LastValue = currentVal
	state.LastTime = now

	// 写入 Tag
	data.SetTag(fmt.Sprintf("delta.%s.current", a.Key), fmt.Sprintf("%.3f", currentVal))
	data.SetTag(fmt.Sprintf("delta.%s.accumulated", a.Key), fmt.Sprintf("%.3f", state.Accumulated))
	data.SetTag(fmt.Sprintf("delta.%s.period", a.Key), a.Period)

	sd.StateStore().Set(stateKey, &state)
	return nil
}

func (a *DeltaAccumulatorAction) ID() string   { return a.IDValue }
func (a *DeltaAccumulatorAction) Type() string { return "delta_accumulator" }
func (a *DeltaAccumulatorAction) Description() string {
	return fmt.Sprintf("delta accumulator key=%s period=%s", a.Key, a.Period)
}

// ─────────────────────────────────────────────
//  EfficiencyCalcAction — 转换效率计算动作
// ─────────────────────────────────────────────

// EfficiencyCalcAction 转换效率计算动作
type EfficiencyCalcAction struct {
	IDValue         string `json:"id"`
	InputPoint      string `json:"input_point"`
	OutputPoint     string `json:"output_point"`
	EfficiencyPoint string `json:"efficiency_point"`
	OutputUnit      string `json:"output_unit"`
}

// NewEfficiencyCalcAction 创建转换效率计算动作
func NewEfficiencyCalcAction(id, inputPoint, outputPoint, efficiencyPoint, outputUnit string) *EfficiencyCalcAction {
	if outputUnit == "" {
		outputUnit = "percent"
	}
	return &EfficiencyCalcAction{
		IDValue:         id,
		InputPoint:      inputPoint,
		OutputPoint:     outputPoint,
		EfficiencyPoint: efficiencyPoint,
		OutputUnit:      outputUnit,
	}
}

func (a *EfficiencyCalcAction) Execute(ctx context.Context, data core.DataContext) error {
	mdc, ok := data.(types.MultiDataContextInterface)
	if !ok {
		return fmt.Errorf("efficiency_calc requires MultiDataContext")
	}

	inputVal, errI := mdc.GetPoint(a.InputPoint)
	outputVal, errO := mdc.GetPoint(a.OutputPoint)
	if errI != nil || errO != nil {
		return fmt.Errorf("efficiency_calc: point not found: input=%v output=%v", errI, errO)
	}

	if inputVal <= 0 {
		data.SetTag(a.EfficiencyPoint, "0")
		return nil
	}

	efficiency := outputVal / inputVal
	switch a.OutputUnit {
	case "percent":
		efficiency *= 100
	}

	data.SetTag(a.EfficiencyPoint, fmt.Sprintf("%.2f", efficiency))
	data.SetTag(a.EfficiencyPoint+".unit", a.OutputUnit)
	return nil
}

func (a *EfficiencyCalcAction) ID() string   { return a.IDValue }
func (a *EfficiencyCalcAction) Type() string { return "efficiency_calc" }
func (a *EfficiencyCalcAction) Description() string {
	return fmt.Sprintf("efficiency calc %s/%s", a.InputPoint, a.OutputPoint)
}

// ─────────────────────────────────────────────
//  PlantSplitAction — 电场/方阵数据拆分动作
// ─────────────────────────────────────────────

// PlantSplitAction 电场/方阵数据拆分动作
type PlantSplitAction struct {
	IDValue      string   `json:"id"`
	SplitBy      string   `json:"split_by"`
	KeySource    string   `json:"key_source"`
	OutputRoutes []string `json:"output_routes"`
}

// NewPlantSplitAction 创建电场/方阵数据拆分动作
func NewPlantSplitAction(id, splitBy, keySource string, outputRoutes []string) *PlantSplitAction {
	return &PlantSplitAction{
		IDValue:      id,
		SplitBy:      splitBy,
		KeySource:    keySource,
		OutputRoutes: outputRoutes,
	}
}

func (a *PlantSplitAction) Execute(ctx context.Context, data core.DataContext) error {
	// 1. 提取拆分键
	var splitKey string
	switch a.KeySource {
	case "device_id":
		splitKey = data.DeviceID()
	case "point_name":
		splitKey = data.PointName()
	default:
		splitKey = data.GetTag(a.KeySource)
	}

	if splitKey == "" {
		return nil
	}

	// 2. 匹配输出路由
	matched := false
	for _, route := range a.OutputRoutes {
		if strings.HasPrefix(splitKey, route) || route == "*" {
			data.AddTarget(route)
			matched = true
		}
	}

	if !matched && len(a.OutputRoutes) > 0 {
		data.AddTarget(a.OutputRoutes[len(a.OutputRoutes)-1])
	}

	data.SetTag("_split_by", a.SplitBy)
	data.SetTag("_split_key", splitKey)
	return nil
}

func (a *PlantSplitAction) ID() string          { return a.IDValue }
func (a *PlantSplitAction) Type() string        { return "plant_split" }
func (a *PlantSplitAction) Description() string { return fmt.Sprintf("plant split by %s", a.SplitBy) }
