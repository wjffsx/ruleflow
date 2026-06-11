package engine

import (
	"context"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  Engine.Prewarm / PrewarmAll
// ─────────────────────────────────────────────

// V13 架构边界说明：
//   - 本文件属于核心库，提供 Engine 的预热功能
//   - 只依赖 core 包，符合核心库零依赖要求
//   - Prewarm/PrewarmAll 是 Engine 的核心方法，不应移出
//
// 使用场景：
//   - 启动预热：减少首次评估的延迟
//   - 规则链预热：提前初始化规则状态

// PrewarmResult 单条规则预热结果
type PrewarmResult struct {
	ChainID  string `json:"chain_id"`
	RuleID   string `json:"rule_id"`
	Duration int64  `json:"duration_ns"`
	Err      error  `json:"-"` // 不序列化
}

// PrewarmSummary 整链预热汇总
type PrewarmSummary struct {
	ChainID   string          `json:"chain_id"`
	RuleCount int             `json:"rule_count"`
	Success   int             `json:"success"`
	Skipped   int             `json:"skipped"`
	Failed    int             `json:"failed"`
	Duration  int64           `json:"duration_ns"`
	Results   []PrewarmResult `json:"results,omitempty"`
}

// Prewarm 对指定链进行预热。
func (e *Engine) Prewarm(ctx context.Context, chainID string) (*PrewarmSummary, error) {
	if e.shutdown.isShuttingDown() {
		return nil, core.NewShutdownError(core.ErrEngineShutdown)
	}

	snap := e.snapshot.Load()
	chain, ok := snap.chains[chainID]
	if !ok {
		return nil, core.NewConfigError(chainID, core.ErrChainNotFound)
	}

	summary := &PrewarmSummary{
		ChainID:   chainID,
		RuleCount: len(chain.SortedRules),
		Results:   make([]PrewarmResult, 0, len(chain.SortedRules)),
	}

	start := time.Now()
	for _, cr := range chain.SortedRules {
		if ctx.Err() != nil {
			summary.Failed = summary.RuleCount - summary.Success - summary.Skipped
			summary.Duration = time.Since(start).Nanoseconds()
			return summary, ctx.Err()
		}

		ruleStart := time.Now()
		err := e.prewarmRule(ctx, cr)
		d := time.Since(ruleStart)

		result := PrewarmResult{
			ChainID:  chainID,
			RuleID:   cr.Rule.ID,
			Duration: d.Nanoseconds(),
			Err:      err,
		}

		switch {
		case err == nil:
			summary.Success++
		case isPrewarmNotImplemented(err):
			summary.Skipped++
			result.Err = nil
		default:
			summary.Failed++
			e.logger.Warn("prewarm rule failed",
				"chain_id", chainID, "rule_id", cr.Rule.ID, "err", err)
		}
		summary.Results = append(summary.Results, result)
	}
	summary.Duration = time.Since(start).Nanoseconds()

	e.logger.Info("chain prewarmed",
		"chain_id", chainID,
		"rules", summary.RuleCount,
		"success", summary.Success,
		"skipped", summary.Skipped,
		"failed", summary.Failed,
		"duration_ns", summary.Duration)
	return summary, nil
}

// PrewarmAll 对所有已加载链进行预热
func (e *Engine) PrewarmAll(ctx context.Context) (map[string]*PrewarmSummary, error) {
	ids := e.GetChainIDs()
	results := make(map[string]*PrewarmSummary, len(ids))
	for _, id := range ids {
		summary, err := e.Prewarm(ctx, id)
		results[id] = summary
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// prewarmRule 预热单条规则
func (e *Engine) prewarmRule(ctx context.Context, cr *core.CompiledRule) error {
	if cr.PrewarmFunc == nil {
		return errPrewarmNotImplemented
	}
	return cr.PrewarmFunc(ctx)
}

// errPrewarmNotImplemented 标识对象未实现 Prewarmable（不视为错误）
var errPrewarmNotImplemented = &prewarmNotImplErr{}

type prewarmNotImplErr struct{}

func (*prewarmNotImplErr) Error() string { return "prewarm not implemented" }

func isPrewarmNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*prewarmNotImplErr)
	return ok
}
