// Package action provides builtin action nodes registration
//
// V7 Refactoring: Updated to use nodes/util instead of builtin/internal/util.
package action

import (
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// GetFactories 返回所有内置动作节点的工厂函数
func GetFactories() map[string]func(id string, config map[string]any) (core.Action, error) {
	return map[string]func(id string, config map[string]any) (core.Action, error){
		"transform":    newTransformAction,
		"rename":       newRenameAction,
		"tag":          newTagAction,
		"drop":         newDropAction,
		"route":        newRouteAction,
		"limit_check":  newLimitCheckAction,
		"quality_mark": newQualityMarkAction,
		"alarm_notify": newAlarmNotifyAction,
		"delay":        newDelayAction,
		// 新增节点
		"bit_unpack": newBitUnpackAction,
		"bit_pack":   newBitPackAction,
	}
}

// ─────────────────────────────────────────────
//  工厂函数
// ─────────────────────────────────────────────

func newTransformAction(id string, config map[string]any) (core.Action, error) {
	var scale, offset *float64
	if v, ok := config["scale"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			scale = &f
		}
	}
	if v, ok := config["offset"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			offset = &f
		}
	}
	unit, _ := config["unit"].(string)
	return NewTransformAction(id, scale, offset, unit), nil
}

func newRenameAction(id string, config map[string]any) (core.Action, error) {
	name, _ := config["point_name"].(string)
	if name == "" {
		return nil, fmt.Errorf("point_name is required")
	}
	return NewRenameAction(id, name), nil
}

func newTagAction(id string, config map[string]any) (core.Action, error) {
	tags := make(map[string]string)
	if raw, ok := config["tags"].(map[string]any); ok {
		for k, v := range raw {
			tags[k], _ = v.(string)
		}
	}
	return NewTagAction(id, tags), nil
}

func newDropAction(id string, config map[string]any) (core.Action, error) {
	return NewDropAction(id), nil
}

func newRouteAction(id string, config map[string]any) (core.Action, error) {
	targets, ok := config["targets"].([]string)
	if !ok {
		if raw, ok := config["targets"].([]any); ok {
			targets = make([]string, len(raw))
			for i, v := range raw {
				targets[i], _ = v.(string)
			}
		}
	}
	return NewRouteAction(id, targets), nil
}

func newLimitCheckAction(id string, config map[string]any) (core.Action, error) {
	return NewLimitCheckAction(id), nil
}

func newQualityMarkAction(id string, config map[string]any) (core.Action, error) {
	good := 0
	bad := 1
	if v, ok := config["good_value"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			good = int(f)
		}
	}
	if v, ok := config["bad_value"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			bad = int(f)
		}
	}
	return NewQualityMarkAction(id, good, bad), nil
}

func newAlarmNotifyAction(id string, config map[string]any) (core.Action, error) {
	severity, _ := config["severity"].(string)
	if severity == "" {
		severity = "warning"
	}
	message, _ := config["message"].(string)
	return NewAlarmNotifyAction(id, severity, message, nil), nil
}

func newDelayAction(id string, config map[string]any) (core.Action, error) {
	durationStr, _ := config["duration"].(string)
	if durationStr == "" {
		return nil, fmt.Errorf("duration is required")
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", durationStr, err)
	}
	return NewDelayAction(id, duration, nil), nil
}

// ─────────────────────────────────────────────
//  新增节点工厂函数
// ─────────────────────────────────────────────

func newBitUnpackAction(id string, config map[string]any) (core.Action, error) {
	var outputTags []string
	if raw, ok := config["output_tags"].([]any); ok {
		outputTags = make([]string, len(raw))
		for i, v := range raw {
			outputTags[i], _ = v.(string)
		}
	}
	if len(outputTags) == 0 {
		return nil, fmt.Errorf("output_tags is required and must be a string array")
	}
	startBit := 0
	if v, ok := config["start_bit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			startBit = int(f)
		}
	}
	return NewBitUnpackAction(id, outputTags, startBit), nil
}

func newBitPackAction(id string, config map[string]any) (core.Action, error) {
	var inputTags []string
	if raw, ok := config["input_tags"].([]any); ok {
		inputTags = make([]string, len(raw))
		for i, v := range raw {
			inputTags[i], _ = v.(string)
		}
	}
	if len(inputTags) == 0 {
		return nil, fmt.Errorf("input_tags is required and must be a string array")
	}
	outputField, _ := config["output_field"].(string)
	if outputField == "" {
		outputField = "value"
	}
	startBit := 0
	if v, ok := config["start_bit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			startBit = int(f)
		}
	}
	return NewBitPackAction(id, inputTags, outputField, startBit), nil
}
