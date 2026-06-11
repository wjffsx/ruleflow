// Package action provides VPP action nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package action

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  AggregatorAction — 聚合计算动作
// ─────────────────────────────────────────────

// GroupedPoint 分组数据点
type GroupedPoint struct {
	DeviceID  string
	PointName string
	Value     float64
	Quality   int
}

// AggregatorAction 聚合计算动作
type AggregatorAction struct {
	IDValue     string   `json:"id"`
	GroupKey    string   `json:"group_key"`    // 聚合分组字段
	Method      string   `json:"method"`       // sum / avg / min / max / count
	InputPoints []string `json:"input_points"` // 输入数据点名
	OutputPoint string   `json:"output_point"` // 聚合结果输出点名
	WindowSec   int      `json:"window_sec"`   // 聚合窗口（秒）
}

// NewAggregatorAction 创建聚合计算动作
func NewAggregatorAction(id, groupKey, method string, inputPoints []string, outputPoint string, windowSec int) *AggregatorAction {
	if method == "" {
		method = "sum"
	}
	return &AggregatorAction{
		IDValue:     id,
		GroupKey:    groupKey,
		Method:      method,
		InputPoints: inputPoints,
		OutputPoint: outputPoint,
		WindowSec:   windowSec,
	}
}

func (a *AggregatorAction) Execute(ctx context.Context, data core.DataContext) error {
	// 获取分组数据（从 MultiDataContext 或 Tag）
	var values []float64

	// 从 Tag 中读取聚合数据
	groupDataTag := data.GetTag("_group_data")
	if groupDataTag != "" {
		// 解析 JSON 格式的分组数据
		var points []GroupedPoint
		if err := json.Unmarshal([]byte(groupDataTag), &points); err == nil {
			for _, pt := range points {
				values = append(values, pt.Value)
			}
		}
	}

	// 从 MultiDataContext 获取
	if mdc, ok := data.(types.MultiDataContextInterface); ok && len(a.InputPoints) > 0 {
		for _, ptName := range a.InputPoints {
			v, err := mdc.GetPoint(ptName)
			if err == nil {
				values = append(values, v)
			}
		}
	}

	if len(values) == 0 {
		return nil
	}

	var result float64
	switch a.Method {
	case "sum":
		for _, v := range values {
			result += v
		}
	case "avg":
		for _, v := range values {
			result += v
		}
		result /= float64(len(values))
	case "min":
		result = values[0]
		for _, v := range values[1:] {
			if v < result {
				result = v
			}
		}
	case "max":
		result = values[0]
		for _, v := range values[1:] {
			if v > result {
				result = v
			}
		}
	case "count":
		result = float64(len(values))
	}

	data.SetTag("_agg_output", a.OutputPoint)
	data.SetTag("_agg_value", fmt.Sprintf("%f", result))
	data.SetTag("_agg_count", fmt.Sprintf("%d", len(values)))
	data.SetTag("_agg_method", a.Method)
	data.SetValue(result)
	return nil
}

func (a *AggregatorAction) ID() string   { return a.IDValue }
func (a *AggregatorAction) Type() string { return "aggregator" }
func (a *AggregatorAction) Description() string {
	return fmt.Sprintf("aggregator %s by %s", a.Method, a.GroupKey)
}