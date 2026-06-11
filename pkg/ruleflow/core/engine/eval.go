package engine

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// evalChain 评估规则链（核心热路径）。
func (e *Engine) evalChain(ctx context.Context, chainID string, data core.DataContext) (result *EvalResult, err error) {
	// 优雅关闭检查
	if e.shutdown.isShuttingDown() {
		return &EvalResult{Data: data}, core.NewShutdownError(core.ErrEngineShutdown)
	}
	if !e.shutdown.begin() {
		return &EvalResult{Data: data}, core.NewShutdownError(core.ErrEngineShutdown)
	}
	defer e.shutdown.end()

	// 评估结果分类（matched/unmatched/dropped/errored）
	evalResult := "unmatched"
	start := time.Now()

	// 指标 defer：最先注册 → 最后执行
	defer func() {
		d := time.Since(start)
		e.metricsSink.ObserveEvalDuration(chainID, d)
		e.metricsSink.IncEvalTotal(chainID, evalResult)
	}()

	// 活跃评估计数 defer
	defer func() {
		n := atomic.AddInt64(&e.activeEval, -1)
		e.metricsSink.SetActiveEval(n)
	}()

	// Panic recovery defer
	defer func() {
		if r := recover(); r != nil {
			panicErr := core.NewPanicError(chainID, "", r)
			e.logger.Error("panic in eval chain", "chain_id", chainID, "err", panicErr)
			e.metricsSink.IncPanic(chainID, "")
			action := e.consultErrorHandler(ctx, chainID, "", "", panicErr, 1)
			if action == core.ErrorActionAbort {
				e.logger.Error("eval chain panic recovered, handler requested abort",
					"chain_id", chainID, "panic", r, "stack", panicErr.Stack)
				err = panicErr
				result = &EvalResult{
					Data:    data,
					Errors:  []RuleError{{Message: panicErr.Error()}},
					Dropped: false,
				}
				evalResult = "errored"
				return
			}
			e.logger.Error("eval chain panic recovered",
				"chain_id", chainID, "panic", r, "stack", panicErr.Stack)
			err = panicErr
			result = &EvalResult{
				Data:    data,
				Errors:  []RuleError{{Message: panicErr.Error()}},
				Dropped: false,
			}
			evalResult = "errored"
		}
	}()

	// 背压级别一次性求值
	bpLevel := contract.Normal
	if e.bpIndicator != nil {
		bpLevel = e.bpIndicator.CurrentLevel()
	}
	hasLossTracker := e.lossTracker != nil

	// 背压检查
	switch bpLevel {
	case contract.Dropping:
		e.metricsSink.IncEvalTotal(chainID, "backpressure_dropped")
		if hasLossTracker {
			e.lossTracker.TrackDrop(ctx, data, "", "backpressure_dropping")
		}
		return &EvalResult{Data: data}, nil
	case contract.Paused:
	case contract.Degraded:
	}

	// 调试开关一次性求值
	debugEnabled := e.debugMgr != nil && e.debugMgr.Enabled()

	// 零锁读取 COW 快照
	snap := e.snapshot.Load()
	chain, ok := snap.chains[chainID]
	if !ok {
		return &EvalResult{Data: data}, nil
	}

	// 追踪
	if e.tracer != nil {
		var end func()
		ctx, end = e.tracer.Begin(ctx, "ruleflow.EvalChain")
		defer end()
	}

	// 评估总超时
	if e.evalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.evalTimeout)
		defer cancel()
	}

	// 如果有 StateStore，包装 DataContext
	evalData := data
	if e.stateStore != nil {
		evalData = WrapWithStateStore(data, e.stateStore)
	}

	// 活跃评估计数
	atomic.AddInt64(&e.activeEval, 1)
	e.metricsSink.SetActiveEval(atomic.LoadInt64(&e.activeEval))

	result = &EvalResult{
		MatchedRules: make([]*core.Rule, 0, len(chain.SortedRules)),
	}

	// 按优先级评估规则
	for _, cr := range chain.SortedRules {
		if ctx.Err() != nil {
			break
		}

		// 背压策略
		if bpLevel == contract.Degraded {
			if cr.Rule.Priority > 10 {
				continue
			}
		}
		if bpLevel == contract.Paused {
			if cr.Rule.Priority > 0 {
				continue
			}
		}

		// 限流器
		if !e.limiter.Allow(contract.LimiterKeyForRule(chainID, cr.Rule.ID)) {
			e.metricsSink.IncRuleThrottled(chainID, cr.Rule.ID)
			if hasLossTracker {
				e.lossTracker.TrackDrop(ctx, evalData, cr.Rule.ID, "throttled")
			}
			continue
		}

		// 调试事件
		if debugEnabled {
			e.debugMgr.CaptureIn(ctx, chainID, cr.Rule.ID, cr.Rule.ID, "condition", "")
		}

		condStart := time.Now()
		condMatched := cr.EvaluateFunc(ctx, evalData)
		condDuration := time.Since(condStart)

		if debugEnabled {
			relationType := "unmatched"
			if condMatched {
				relationType = "matched"
			}
			e.debugMgr.CaptureOut(ctx, chainID, cr.Rule.ID, cr.Rule.ID, "condition", relationType, "", condDuration.Nanoseconds(), "")
		}

		e.metricsSink.ObserveConditionEval(chainID, cr.Rule.ID, condDuration, condMatched)

		if !condMatched {
			continue
		}

		result.MatchedRules = append(result.MatchedRules, cr.Rule)

		if cr.ExecuteFunc != nil {
			if debugEnabled {
				e.debugMgr.CaptureIn(ctx, chainID, cr.Rule.ID, cr.Rule.ID, "action", "")
			}

			actionStart := time.Now()
			execErr := e.executeWithGuard(ctx, cr, evalData)
			actionDuration := time.Since(actionStart)

			if execErr != nil {
				if debugEnabled {
					e.debugMgr.CaptureOut(ctx, chainID, cr.Rule.ID, cr.Rule.ID, "action", "error", "", actionDuration.Nanoseconds(), execErr.Error())
				}
			} else {
				if debugEnabled {
					e.debugMgr.CaptureOut(ctx, chainID, cr.Rule.ID, cr.Rule.ID, "action", "matched", "", actionDuration.Nanoseconds(), "")
				}
			}

			hasActionErr := execErr != nil
			e.metricsSink.ObserveActionExec(chainID, cr.Rule.ID, "action", actionDuration, hasActionErr)

			if execErr != nil {
				if execErr == core.ErrDropData {
					evalData.SetDropped(true)
					e.metricsSink.IncActionTotal(chainID, "drop", "ok")
					if hasLossTracker {
						e.lossTracker.TrackDrop(ctx, evalData, cr.Rule.ID, "drop_action")
					}
					result.Data = evalData
					result.Dropped = true
					result.Duration = time.Since(start)
					e.savePreviousValue(evalData)
					if e.evalHook != nil {
						e.evalHook.OnEval(ctx, chainID, evalData, result)
					}
					evalResult = "dropped"
					return result, nil
				}

				e.metricsSink.IncActionTotal(chainID, "rule", "errored")
				structErr := core.NewActionError(chainID, cr.Rule.ID, "", execErr)
				e.logger.Error("action exec error", "chain_id", chainID, "rule_id", cr.Rule.ID, "err", structErr)
				result.Errors = append(result.Errors, RuleError{
					RuleID:  cr.Rule.ID,
					Message: structErr.Error(),
				})

				if hasLossTracker {
					e.lossTracker.TrackError(ctx, evalData, cr.Rule.ID, execErr)
				}

				action := e.consultErrorHandler(ctx, chainID, cr.Rule.ID, "", structErr, 1)
				if action == core.ErrorActionRetry {
					execErr = e.executeWithGuard(ctx, cr, evalData)
					if execErr == nil {
						e.metricsSink.IncActionTotal(chainID, "rule", "ok")
						for _, t := range cr.Rule.Targets {
							evalData.AddTarget(t)
						}
						if chain.EvalMode == core.EvalModeFirst {
							break
						}
						continue
					}
				} else if action == core.ErrorActionFallback {
					if e.fallbackFunc != nil {
						if fbErr := e.fallbackFunc(ctx, chainID, cr.Rule.ID, evalData); fbErr != nil {
							e.logger.Warn("fallback action failed",
								"chain_id", chainID, "rule_id", cr.Rule.ID, "err", fbErr)
						} else {
							e.metricsSink.IncActionTotal(chainID, "rule", "ok")
							for _, t := range cr.Rule.Targets {
								evalData.AddTarget(t)
							}
							if chain.EvalMode == core.EvalModeFirst {
								break
							}
							continue
						}
					}
				}

				if action == core.ErrorActionAbort || (action == core.ErrorActionContinue && chain.ErrorStrategy == core.ErrorStrategyAbort) {
					result.Data = evalData
					result.Duration = time.Since(start)
					e.savePreviousValue(evalData)
					if e.evalHook != nil {
						e.evalHook.OnEval(ctx, chainID, evalData, result)
					}
					evalResult = "errored"
					return result, structErr
				}
				continue
			}
			e.metricsSink.IncActionTotal(chainID, "rule", "ok")
		}

		for _, t := range cr.Rule.Targets {
			evalData.AddTarget(t)
		}

		if chain.EvalMode == core.EvalModeFirst {
			break
		}

		if evalData.Dropped() {
			if hasLossTracker {
				e.lossTracker.TrackDrop(ctx, evalData, cr.Rule.ID, "marked_dropped")
			}
			result.Data = evalData
			result.Dropped = true
			result.Duration = time.Since(start)
			e.savePreviousValue(evalData)
			if e.evalHook != nil {
				e.evalHook.OnEval(ctx, chainID, evalData, result)
			}
			evalResult = "dropped"
			return result, nil
		}
	}

	result.Data = evalData
	result.Duration = time.Since(start)

	e.savePreviousValue(evalData)

	if e.evalHook != nil {
		e.evalHook.OnEval(ctx, chainID, evalData, result)
	}

	if len(result.MatchedRules) > 0 {
		evalResult = "matched"
	}
	return result, nil
}

// consultErrorHandler 咨询 ErrorHandler 的下一步动作。
func (e *Engine) consultErrorHandler(ctx context.Context, chainID, ruleID, actionID string, rfe *core.RuleFlowError, attempt int) core.ErrorAction {
	if e.errorHandler == nil {
		return core.ErrorActionContinue
	}
	hctx := core.HandlerContext{
		ChainID:   chainID,
		RuleID:    ruleID,
		ActionID:  actionID,
		ErrorType: rfe.Type,
		Attempt:   attempt,
	}
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("error handler panic recovered",
				"chain_id", chainID, "rule_id", ruleID, "panic", r)
		}
	}()
	action, herr := e.errorHandler.HandleError(hctx, rfe)
	if herr != nil {
		e.logger.Warn("error handler returned error",
			"chain_id", chainID, "rule_id", ruleID, "err", herr)
	}
	return action
}

// savePreviousValue 保存当前值作为下次评估的前值。
func (e *Engine) savePreviousValue(data core.DataContext) {
	data.SetPreviousValue(data.Value())
}

// executeWithGuard 动作执行保护（超时 + panic recovery）。
func (e *Engine) executeWithGuard(ctx context.Context, cr *core.CompiledRule, data core.DataContext) (err error) {
	if e.actionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.actionTimeout)
		defer cancel()
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := core.NewPanicError(cr.Rule.ID, "", r)
			e.logger.Error("action panic recovered",
				"rule_id", cr.Rule.ID, "panic", r, "stack", panicErr.Stack)
			err = panicErr
		}
	}()

	return cr.ExecuteFunc(ctx, data)
}
