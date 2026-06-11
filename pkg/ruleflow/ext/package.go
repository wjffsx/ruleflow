package ext

import (
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes/util"
)

// ─────────────────────────────────────────────
//  extPackage — NodePackage 实现
// ─────────────────────────────────────────────

// extPackage 实现 nodes.NodePackage 接口
type extPackage struct{}

// Package 是 ext 包的 NodePackage 单例
var Package nodes.NodePackage = extPackage{}

// GetConditionFactories 返回所有自定义 Condition 的工厂映射
func (extPackage) GetConditionFactories() map[string]nodes.ConditionFactory {
	return map[string]nodes.ConditionFactory{
		"expr_filter": func(id string, config map[string]any) (core.Condition, error) {
			exprStr, _ := config["expr"].(string)
			return NewExprFilterCondition(id, exprStr), nil
		},
		"historical_compare": func(id string, config map[string]any) (core.Condition, error) {
			ago, _ := config["ago"].(string)
			operator, _ := config["operator"].(string)
			threshold := 0.1
			if v, ok := config["threshold"]; ok {
				if f, ok := v.(float64); ok {
					threshold = f
				}
			}
			return NewHistoricalCompareCondition(id, ago, operator, threshold, nil), nil
		},
	}
}

// GetActionFactories 返回所有自定义 Action 的工厂映射
func (extPackage) GetActionFactories() map[string]nodes.ActionFactory {
	return map[string]nodes.ActionFactory{
		"alarm_notify_ext": func(id string, config map[string]any) (core.Action, error) {
			alarmType, _ := config["alarmType"].(string)
			severity, _ := config["severity"].(string)
			action := NewAlarmNotifyExtAction(id, alarmType, severity, nil)
			return action, nil
		},
		"quality_mark_ext": func(id string, config map[string]any) (core.Action, error) {
			quality, _ := config["quality"].(string)
			action := NewQualityMarkExtAction(id, quality, nil)
			return action, nil
		},
		"calc_node": func(id string, config map[string]any) (core.Action, error) {
			formula, _ := config["formula"].(string)
			action := NewCalcNodeAction(id, formula, nil, "")
			return action, nil
		},
		"storage_write": func(id string, config map[string]any) (core.Action, error) {
			target, _ := config["target"].(string)
			action := NewStorageWriteAction(id, target, nil, nil)
			return action, nil
		},
		"aggregation_write": func(id string, _ map[string]any) (core.Action, error) {
			return NewAggregationWriteAction(id, nil), nil
		},
		"device_aggregator": func(id string, config map[string]any) (core.Action, error) {
			inputPoint, _ := config["input_point"].(string)
			var mappings []OutputMapping
			if v, ok := config["output_mappings"]; ok {
				if arr, ok := v.([]any); ok {
					for _, item := range arr {
						if m, ok := item.(map[string]any); ok {
							mapping := OutputMapping{}
							mapping.Category, _ = m["category"].(string)
							mapping.Output, _ = m["output"].(string)
							mapping.Target, _ = m["target"].(string)
							mappings = append(mappings, mapping)
						}
					}
				}
			}
			return NewDeviceAggregateAction(id, inputPoint, mappings, nil, nil), nil
		},
		"status_change_log": func(id string, _ map[string]any) (core.Action, error) {
			return NewStatusChangeLogAction(id, nil, nil), nil
		},
		"expr_switch": func(id string, config map[string]any) (core.Action, error) {
			expr, _ := config["expression"].(string)
			return NewExprSwitchAction(id, expr), nil
		},
		"multi_device_control": func(id string, config map[string]any) (core.Action, error) {
			var targets []string
			if raw, ok := config["targets"].([]any); ok {
				for _, v := range raw {
					if s, ok := v.(string); ok {
						targets = append(targets, s)
					}
				}
			}
			command, _ := config["command"].(string)
			params := make(map[string]any)
			if raw, ok := config["params"].(map[string]any); ok {
				params = raw
			}
			return NewMultiDeviceControlAction(id, targets, command, params, nil), nil
		},
		"strategy_execute": func(id string, config map[string]any) (core.Action, error) {
			strategyID, _ := config["strategy_id"].(string)
			params := make(map[string]any)
			if raw, ok := config["params"].(map[string]any); ok {
				params = raw
			}
			return NewStrategyExecuteAction(id, strategyID, params, nil), nil
		},
		// 新增节点
		"emit_soe": func(id string, config map[string]any) (core.Action, error) {
			eventType, _ := config["event_type"].(string)
			severity, _ := config["severity"].(string)
			descTemplate, _ := config["description_template"].(string)
			return NewEmitSOEAction(id, eventType, severity, descTemplate, nil), nil
		},
		"limit_tracker": func(id string, config map[string]any) (core.Action, error) {
			trackDuration := false
			if v, ok := config["track_duration"]; ok {
				trackDuration = v == true || v == "true"
			}
			trackCount := false
			if v, ok := config["track_count"]; ok {
				trackCount = v == true || v == "true"
			}
			hysteresis := 0.0
			if v, ok := config["hysteresis"]; ok {
				if f, ok := util.ToFloat64(v); ok {
					hysteresis = f
				}
			}
			return NewLimitTrackerAction(id, trackDuration, trackCount, hysteresis, nil), nil
		},
		"meter_freeze": func(id string, config map[string]any) (core.Action, error) {
			freezeType, _ := config["freeze_type"].(string)
			periodStr, _ := config["period"].(string)
			var period time.Duration
			if periodStr != "" {
				period, _ = time.ParseDuration(periodStr)
			}
			return NewMeterFreezeAction(id, freezeType, period, nil), nil
		},
		"demand_calc": func(id string, config map[string]any) (core.Action, error) {
			periodStr, _ := config["period"].(string)
			var period time.Duration
			if periodStr != "" {
				period, _ = time.ParseDuration(periodStr)
			}
			method, _ := config["method"].(string)
			intervalStr, _ := config["interval"].(string)
			var interval time.Duration
			if intervalStr != "" {
				interval, _ = time.ParseDuration(intervalStr)
			}
			return NewDemandCalcAction(id, period, method, interval, nil), nil
		},
	}
}
