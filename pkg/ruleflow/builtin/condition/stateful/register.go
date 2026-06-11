// Package stateful provides stateful condition nodes registration
//
// V7 Refactoring: Updated to use nodes/util instead of builtin/internal/util.
package stateful

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// GetFactories 返回所有有状态条件节点的工厂函数
func GetFactories() map[string]func(id string, config map[string]any) (core.Condition, error) {
	return map[string]func(id string, config map[string]any) (core.Condition, error){
		"state_change":      newStateChangeCondition,
		"duration":          newDurationCondition,
		"trend":             newTrendCondition,
		"periodic":          newPeriodicCondition,
		"dynamic_threshold": newDynamicThresholdCondition,
		// 新增节点
		"rate_limit_window": newRateLimitWindowCondition,
		"limit_recovery":    newLimitRecoveryCondition,
	}
}

// ─────────────────────────────────────────────
//  工厂函数
// ─────────────────────────────────────────────

func newStateChangeCondition(id string, config map[string]any) (core.Condition, error) {
	var from, to *float64
	if v, ok := config["from"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			from = &f
		}
	}
	if v, ok := config["to"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			to = &f
		}
	}
	return NewStateChangeCondition(id, from, to), nil
}

func newDurationCondition(id string, config map[string]any) (core.Condition, error) {
	durationStr, _ := config["duration"].(string)
	if durationStr == "" {
		return nil, fmt.Errorf("duration is required")
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", durationStr, err)
	}
	// duration 条件需要内嵌条件，这里创建一个占位条件
	// 实际使用时由 config/parse 递归解析替换
	inner := newPlaceholderCondition(id + "_inner")
	return NewDurationCondition(id, inner, duration), nil
}

func newTrendCondition(id string, config map[string]any) (core.Condition, error) {
	direction, _ := config["direction"].(string)
	if direction == "" {
		direction = "increasing"
	}
	windowStr, _ := config["window"].(string)
	if windowStr == "" {
		windowStr = "5m"
	}
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("invalid window %q: %w", windowStr, err)
	}
	threshold := 0.1
	if v, ok := config["threshold"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			threshold = f
		}
	}
	return NewTrendCondition(id, direction, window, threshold), nil
}

func newPeriodicCondition(id string, config map[string]any) (core.Condition, error) {
	periodStr, _ := config["period"].(string)
	if periodStr == "" {
		periodStr = "1m"
	}
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		return nil, fmt.Errorf("invalid period %q: %w", periodStr, err)
	}
	count := 3
	if v, ok := config["count"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			count = int(f)
		}
	}
	inner := newPlaceholderCondition(id + "_inner")
	return NewPeriodicCondition(id, inner, period, count), nil
}

func newDynamicThresholdCondition(id string, config map[string]any) (core.Condition, error) {
	operator, _ := config["operator"].(string)
	if operator == "" {
		operator = "gt"
	}
	source, _ := config["source"].(string)
	if source == "" {
		return nil, fmt.Errorf("source is required")
	}
	return NewDynamicThresholdCondition(id, operator, source), nil
}

// placeholderCondition 是一个占位条件，始终返回 true。
// 用于 Duration / Periodic 工厂中作为默认的内嵌条件，
// 后续由 config/parse 递归解析替换为实际条件。
type placeholderCondition struct{ id string }

func (p *placeholderCondition) ID() string                                              { return p.id }
func (p *placeholderCondition) Type() string                                            { return "placeholder" }
func (p *placeholderCondition) Description() string                                     { return "placeholder (replaced during config parse)" }
func (p *placeholderCondition) Evaluate(_ context.Context, _ core.DataContext) bool { return true }

func newPlaceholderCondition(id string) core.Condition {
	return &placeholderCondition{id: id}
}

// ─────────────────────────────────────────────
//  新增节点工厂函数
// ─────────────────────────────────────────────

func newRateLimitWindowCondition(id string, config map[string]any) (core.Condition, error) {
	rateThreshold := 0.0
	if v, ok := config["rate_threshold"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			rateThreshold = f
		}
	}
	if rateThreshold == 0 {
		return nil, fmt.Errorf("rate_threshold is required and must be > 0")
	}
	windowStr, _ := config["window"].(string)
	if windowStr == "" {
		windowStr = "5s"
	}
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("invalid window %q: %w", windowStr, err)
	}
	return NewRateLimitWindowCondition(id, rateThreshold, window), nil
}

func newLimitRecoveryCondition(id string, config map[string]any) (core.Condition, error) {
	durationStr, _ := config["duration"].(string)
	if durationStr == "" {
		durationStr = "5s"
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", durationStr, err)
	}
	hysteresis := 0.0
	if v, ok := config["hysteresis"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			hysteresis = f
		}
	}
	return NewLimitRecoveryCondition(id, duration, hysteresis), nil
}
