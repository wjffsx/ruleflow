// Package flow provides VPP flow nodes registration
//
// V7 Refactoring: Updated to use nodes/util instead of extensions/internal/util.
package flow

import (
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
)

// GetFactories 返回所有 VPP 流程节点的工厂函数
func GetFactories() map[string]func(id string, config map[string]any) (core.Action, error) {
	return map[string]func(id string, config map[string]any) (core.Action, error){
		"msg_generator": newMsgGeneratorAction,
		"sub_chain":     newSubChainAction,
	}
}

// ─────────────────────────────────────────────
//  工厂函数
// ─────────────────────────────────────────────

func newMsgGeneratorAction(id string, config map[string]any) (core.Action, error) {
	outputPoint, _ := config["output_point"].(string)
	outputValue := 1.0
	if v, ok := config["output_value"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			outputValue = f
		}
	}
	tags := make(map[string]string)
	if raw, ok := config["tags"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				tags[k] = s
			}
		}
	}
	intervalSec := 60
	if v, ok := config["interval_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			intervalSec = int(f)
		}
	}
	return NewMsgGeneratorAction(id, outputPoint, outputValue, tags, intervalSec), nil
}

func newSubChainAction(id string, config map[string]any) (core.Action, error) {
	chainID, _ := config["chain_id"].(string)
	sync := false
	if v, ok := config["sync"]; ok {
		if b, ok := v.(bool); ok {
			sync = b
		}
	}
	inputMapping := make(map[string]string)
	if raw, ok := config["input_mapping"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				inputMapping[k] = s
			}
		}
	}
	outputMapping := make(map[string]string)
	if raw, ok := config["output_mapping"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				outputMapping[k] = s
			}
		}
	}
	return NewSubChainAction(id, chainID, sync, inputMapping, outputMapping, nil), nil
}
