// Package condition provides builtin condition nodes registration
//
// V7 Refactoring: Updated to use nodes/util instead of builtin/internal/util.
package condition

import (
	"fmt"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/builtin/condition/stateful"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes/util"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// GetFactories 返回所有内置条件节点的工厂函数
func GetFactories() map[string]func(id string, config map[string]any) (core.Condition, error) {
	factories := map[string]func(id string, config map[string]any) (core.Condition, error){
		"device_type":        newDeviceTypeCondition,
		"device_id":          newDeviceIDCondition,
		"point_name":         newPointNameCondition,
		"point_name_pattern": newPointNamePatternCondition,
		"fqn_prefix":         newFQNCondition,
		"value_range":        newValueRangeCondition,
		"value_in":           newValueInCondition,
		"quality":            newQualityCondition,
		"limit_exceeded":     newLimitExceededCondition,
		"time_window":        newTimeWindowCondition,
		// 新增节点
		"bit_mask":        newBitMaskCondition,
		"delta_threshold": newDeltaThresholdCondition,
		"rate_limit":      newRateLimitCondition,
	}
	// 合入 stateful
	for k, v := range stateful.GetFactories() {
		factories[k] = v
	}
	return factories
}

// ─────────────────────────────────────────────
//  工厂函数
// ─────────────────────────────────────────────

func newDeviceTypeCondition(id string, config map[string]any) (core.Condition, error) {
	types, ok := config["device_types"].([]string)
	if !ok {
		// 尝试从 []any 转换
		if raw, ok := config["device_types"].([]any); ok {
			types = make([]string, len(raw))
			for i, v := range raw {
				types[i], _ = v.(string)
			}
		} else {
			return nil, fmt.Errorf("device_types is required and must be a string array")
		}
	}
	return NewDeviceTypeCondition(id, types), nil
}

func newDeviceIDCondition(id string, config map[string]any) (core.Condition, error) {
	ids, ok := config["device_ids"].([]string)
	if !ok {
		if raw, ok := config["device_ids"].([]any); ok {
			ids = make([]string, len(raw))
			for i, v := range raw {
				ids[i], _ = v.(string)
			}
		} else {
			return nil, fmt.Errorf("device_ids is required and must be a string array")
		}
	}
	return NewDeviceIDCondition(id, ids), nil
}

func newPointNameCondition(id string, config map[string]any) (core.Condition, error) {
	names, ok := config["point_names"].([]string)
	if !ok {
		if raw, ok := config["point_names"].([]any); ok {
			names = make([]string, len(raw))
			for i, v := range raw {
				names[i], _ = v.(string)
			}
		} else {
			return nil, fmt.Errorf("point_names is required and must be a string array")
		}
	}
	return NewPointNameCondition(id, names), nil
}

func newPointNamePatternCondition(id string, config map[string]any) (core.Condition, error) {
	pattern, _ := config["pattern"].(string)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	return NewPointNamePatternCondition(id, pattern)
}

func newFQNCondition(id string, config map[string]any) (core.Condition, error) {
	prefixes, ok := config["prefixes"].([]string)
	if !ok {
		if raw, ok := config["prefixes"].([]any); ok {
			prefixes = make([]string, len(raw))
			for i, v := range raw {
				prefixes[i], _ = v.(string)
			}
		} else {
			return nil, fmt.Errorf("prefixes is required and must be a string array")
		}
	}
	return NewFQNCondition(id, prefixes), nil
}

func newValueRangeCondition(id string, config map[string]any) (core.Condition, error) {
	var min, max *float64
	if v, ok := config["min_value"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			min = &f
		}
	}
	if v, ok := config["max_value"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			max = &f
		}
	}
	return NewValueRangeCondition(id, min, max), nil
}

func newValueInCondition(id string, config map[string]any) (core.Condition, error) {
	var values []float64
	if raw, ok := config["values"].([]any); ok {
		for _, v := range raw {
			if f, ok := util.ToFloat64(v); ok {
				values = append(values, f)
			}
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("values is required and must be a number array")
	}
	return NewValueInCondition(id, values), nil
}

func newQualityCondition(id string, config map[string]any) (core.Condition, error) {
	minQ := 0
	if v, ok := config["min_quality"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			minQ = int(f)
		}
	}
	return NewQualityCondition(id, minQ), nil
}

func newLimitExceededCondition(id string, config map[string]any) (core.Condition, error) {
	return NewLimitExceededCondition(id), nil
}

func newTimeWindowCondition(id string, config map[string]any) (core.Condition, error) {
	start, _ := config["start"].(string)
	end, _ := config["end"].(string)
	timezone, _ := config["timezone"].(string)
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	var weekdays []int
	if raw, ok := config["weekdays"].([]any); ok {
		for _, v := range raw {
			if f, ok := util.ToFloat64(v); ok {
				weekdays = append(weekdays, int(f))
			}
		}
	}
	if start == "" || end == "" {
		return nil, fmt.Errorf("start and end are required")
	}
	return NewTimeWindowCondition(id, start, end, timezone, weekdays), nil
}

// ─────────────────────────────────────────────
//  新增节点工厂函数
// ─────────────────────────────────────────────

func newBitMaskCondition(id string, config map[string]any) (core.Condition, error) {
	mask := uint64(0)
	if v, ok := config["mask"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			mask = uint64(f)
		}
	}
	expected := uint64(0)
	if v, ok := config["expected"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			expected = uint64(f)
		}
	}
	operator, _ := config["operator"].(string)
	if operator == "" {
		operator = "and"
	}
	return NewBitMaskCondition(id, mask, expected, operator), nil
}

func newDeltaThresholdCondition(id string, config map[string]any) (core.Condition, error) {
	threshold := 0.0
	if v, ok := config["threshold"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			threshold = f
		}
	}
	if threshold == 0 {
		return nil, fmt.Errorf("threshold is required and must be > 0")
	}
	direction, _ := config["direction"].(string)
	if direction == "" {
		direction = "both"
	}
	return NewDeltaThresholdCondition(id, threshold, direction), nil
}

func newRateLimitCondition(id string, config map[string]any) (core.Condition, error) {
	rateThreshold := 0.0
	if v, ok := config["rate_threshold"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			rateThreshold = f
		}
	}
	if rateThreshold == 0 {
		return nil, fmt.Errorf("rate_threshold is required and must be > 0")
	}
	direction, _ := config["direction"].(string)
	if direction == "" {
		direction = "both"
	}
	return NewRateLimitCondition(id, rateThreshold, direction), nil
}
