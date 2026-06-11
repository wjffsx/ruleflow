package ext

import (
	"context"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  StrategyExecuteAction — 策略执行动作
// ─────────────────────────────────────────────

// StrategyExecutor 策略执行器接口（由服务层实现）
type StrategyExecutor interface {
	Execute(strategyID string, params map[string]any) error
}

// StrategyExecuteAction 策略执行动作
type StrategyExecuteAction struct {
	IDValue    string
	StrategyID string
	Params     map[string]any
	Executor   StrategyExecutor
}

var _ core.Action = (*StrategyExecuteAction)(nil)

func NewStrategyExecuteAction(id, strategyID string, params map[string]any, executor StrategyExecutor) *StrategyExecuteAction {
	return &StrategyExecuteAction{
		IDValue:    id,
		StrategyID: strategyID,
		Params:     params,
		Executor:   executor,
	}
}

func (a *StrategyExecuteAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Executor == nil {
		return nil
	}

	// 合并 DataContext 信息到参数
	params := make(map[string]any, len(a.Params)+4)
	for k, v := range a.Params {
		params[k] = v
	}
	params["device_id"] = data.DeviceID()
	params["point_name"] = data.PointName()
	params["value"] = data.Value()
	params["timestamp"] = data.Timestamp()

	return a.Executor.Execute(a.StrategyID, params)
}

func (a *StrategyExecuteAction) ID() string          { return a.IDValue }
func (a *StrategyExecuteAction) Type() string        { return "strategy_execute" }
func (a *StrategyExecuteAction) Description() string { return "strategy execute" }
