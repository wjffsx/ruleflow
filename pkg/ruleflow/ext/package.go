package ext

import (
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
)

// ─────────────────────────────────────────────
//  extPackage — NodePackage 实现
// ─────────────────────────────────────────────

// extPackage 实现 core.NodePackage 接口
type extPackage struct{}

// 编译期接口检查：extPackage 实现 core.NodePackage
var _ core.NodePackage = extPackage{}

// Package 是 ext 包的 NodePackage 单例
var Package core.NodePackage = extPackage{}

// GetConditionFactories 返回所有自定义 Condition 的工厂映射
func (extPackage) GetConditionFactories() map[string]core.ConditionFactory {
	return map[string]core.ConditionFactory{
		"expr_filter": func(id string, config map[string]any) (core.Condition, error) {
			exprStr, ok := config["expr"].(string)
			if !ok || exprStr == "" {
				return nil, fmt.Errorf("expr_filter: missing or invalid 'expr' config")
			}
			return NewExprFilterCondition(id, exprStr), nil
		},
		"historical_compare": func(id string, config map[string]any) (core.Condition, error) {
			ago, ok := config["ago"].(string)
			if !ok || ago == "" {
				return nil, fmt.Errorf("historical_compare: missing or invalid 'ago' config")
			}
			operator, _ := config["operator"].(string)
			threshold := 0.1
			if v, ok := config["threshold"]; ok {
				if f, ok := v.(float64); ok {
					threshold = f
				}
			}
			return NewHistoricalCompareCondition(id, ago, operator, threshold, nil), nil
		},
		// ── 告警检测条件节点 ──
		"threshold_detect": func(id string, config map[string]any) (core.Condition, error) {
			operator, _ := config["operator"].(string)
			if operator == "" {
				return nil, fmt.Errorf("threshold_detect: missing or invalid 'operator' config")
			}
			value := 0.0
			if v, ok := config["value"]; ok {
				if f, ok := util.ToFloat64(v); ok {
					value = f
				}
			}
			minValue := 0.0
			if v, ok := config["min_value"]; ok {
				if f, ok := util.ToFloat64(v); ok {
					minValue = f
				}
			}
			var duration time.Duration
			if durStr, ok := config["duration"].(string); ok && durStr != "" {
				duration, _ = time.ParseDuration(durStr)
			}
			// StateStore 在运行时通过引擎注入，这里传 nil
			cond := NewThresholdDetectCondition(id, operator, value, duration, nil)
			cond.MinValue = minValue
			return cond, nil
		},
		"status_detect": func(id string, config map[string]any) (core.Condition, error) {
			expectedValue, ok := config["expected_value"]
			if !ok || expectedValue == "" {
				return nil, fmt.Errorf("status_detect: missing or invalid 'expected_value' config")
			}
			var debounce time.Duration
			if durStr, ok := config["debounce"].(string); ok && durStr != "" {
				debounce, _ = time.ParseDuration(durStr)
			}
			return NewStatusDetectCondition(id, expectedValue, debounce, nil), nil
		},
		"rate_detect": func(id string, config map[string]any) (core.Condition, error) {
			maxRate := 0.0
			if v, ok := config["max_rate"]; ok {
				if f, ok := util.ToFloat64(v); ok {
					maxRate = f
				}
			}
			if maxRate <= 0 {
				return nil, fmt.Errorf("rate_detect: missing or invalid 'max_rate' config")
			}
			var window time.Duration
			if windowStr, ok := config["window"].(string); ok && windowStr != "" {
				window, _ = time.ParseDuration(windowStr)
			}
			if window <= 0 {
				window = 60 * time.Second
			}
			direction, _ := config["direction"].(string)
			return NewRateDetectCondition(id, maxRate, window, direction, nil), nil
		},
		"composite_detect": func(id string, config map[string]any) (core.Condition, error) {
			logic, _ := config["logic"].(string)
			if logic == "" {
				logic = "and"
			}
			var conditions []SubCondition
			if raw, ok := config["conditions"].([]any); ok {
				for _, item := range raw {
					if m, ok := item.(map[string]any); ok {
						cond := SubCondition{}
						cond.Operator, _ = m["operator"].(string)
						cond.Field, _ = m["field"].(string)
						if v, ok := m["threshold"]; ok {
							if f, ok := util.ToFloat64(v); ok {
								cond.Threshold = f
							}
						}
						if v, ok := m["min_value"]; ok {
							if f, ok := util.ToFloat64(v); ok {
								cond.MinValue = f
							}
						}
						conditions = append(conditions, cond)
					}
				}
			}
			if len(conditions) == 0 {
				return nil, fmt.Errorf("composite_detect: 'conditions' is required with at least one condition")
			}
			return NewCompositeDetectCondition(id, logic, conditions), nil
		},
	}
}

// GetActionFactories 返回所有自定义 Action 的工厂映射
func (extPackage) GetActionFactories() map[string]core.ActionFactory {
	return map[string]core.ActionFactory{
		"alarm_notify_ext": func(id string, config map[string]any) (core.Action, error) {
			alarmType, ok := config["alarmType"].(string)
			if !ok || alarmType == "" {
				return nil, fmt.Errorf("alarm_notify_ext: missing or invalid 'alarmType' config")
			}
			severity, _ := config["severity"].(string)
			action := NewAlarmNotifyExtAction(id, alarmType, severity, nil)
			return action, nil
		},
		"quality_mark_ext": func(id string, config map[string]any) (core.Action, error) {
			quality, ok := config["quality"].(string)
			if !ok || quality == "" {
				return nil, fmt.Errorf("quality_mark_ext: missing or invalid 'quality' config")
			}
			action := NewQualityMarkExtAction(id, quality, nil)
			return action, nil
		},
		"calc_node": func(id string, config map[string]any) (core.Action, error) {
			formula, ok := config["formula"].(string)
			if !ok || formula == "" {
				return nil, fmt.Errorf("calc_node: missing or invalid 'formula' config")
			}
			action := NewCalcNodeAction(id, formula, nil, "")
			return action, nil
		},
		"storage_write": func(id string, config map[string]any) (core.Action, error) {
			target, ok := config["target"].(string)
			if !ok || target == "" {
				return nil, fmt.Errorf("storage_write: missing or invalid 'target' config")
			}
			action := NewStorageWriteAction(id, target, nil, nil)
			return action, nil
		},
		"aggregation_write": func(id string, _ map[string]any) (core.Action, error) {
			return NewAggregationWriteAction(id, nil), nil
		},
		"device_aggregator": func(id string, config map[string]any) (core.Action, error) {
			inputPoint, ok := config["input_point"].(string)
			if !ok || inputPoint == "" {
				return nil, fmt.Errorf("device_aggregator: missing or invalid 'input_point' config")
			}
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
			expr, ok := config["expression"].(string)
			if !ok || expr == "" {
				return nil, fmt.Errorf("expr_switch: missing or invalid 'expression' config")
			}
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
			strategyID, ok := config["strategy_id"].(string)
			if !ok || strategyID == "" {
				return nil, fmt.Errorf("strategy_execute: missing or invalid 'strategy_id' config")
			}
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
		// ── 告警检测动作节点 ──
		"alarm_emit": func(id string, config map[string]any) (core.Action, error) {
			severity, _ := config["severity"].(string)
			sourceType, _ := config["source_type"].(string)
			messageTemplate, _ := config["message_template"].(string)
			return NewAlarmEmitAction(id, severity, sourceType, messageTemplate), nil
		},
	}
}
