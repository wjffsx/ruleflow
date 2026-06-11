// Package engine 提供轻量规则引擎
//
// 拆分说明（V7 §6.2 任务 6）：
//   - 原 engine.go 单文件 934 行，承担 12+ 职责
//   - 拆分为多个内部文件后，本文件仅作为对外 API 的薄封装层
//   - 内部实现分布在本包内的多个文件中（types / options / load / eval / pool / prewarm / health_shutdown / state）
//
// V12 迁移：移除 debug 包依赖，改用 contract.DebugManager 接口
//
// 公共 API 类型：
//   - Engine / EngineOption / WithXxx — 引擎与配置
//   - EvalResult / RuleError / EvalHook — 评估结果与回调
//   - LoadChain / UnloadChain / EvalChain / EvalChainBatch — 核心操作
//   - Prewarm / PrewarmAll — 预热
//   - HealthCheck / Shutdown / GetChainIDs / GetDebugManager — 运维
//   - AcquireResult / ReleaseResult / WithResultPool — 池化
//   - WrapWithStateStore — DataContext 包装
package engine

import (
	"context"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  顶层 API 透传（Go 不支持函数别名，必须显式包一层）
//  实际实现已下沉至同名小写函数 / 其他文件。
// ─────────────────────────────────────────────

// WithTracer 配置追踪器（透传至 options.go）。
func WithTracer(tp contract.TracerProvider) EngineOption {
	return withTracer(tp)
}

// WithBackpressureIndicator 配置背压指示器。
func WithBackpressureIndicator(bp contract.Indicator) EngineOption {
	return withBackpressureIndicator(bp)
}

// WithDataLossTracker 配置数据丢失追踪器。
func WithDataLossTracker(dlt contract.Tracker) EngineOption {
	return withDataLossTracker(dlt)
}

// WithRegistry 配置注册表。
func WithRegistry(r core.Registry) EngineOption {
	return withRegistry(r)
}

// WithActionTimeout 配置单次 action 执行的超时。
func WithActionTimeout(timeout time.Duration) EngineOption {
	return withActionTimeout(timeout)
}

// WithErrorStrategy 配置错误策略。
func WithErrorStrategy(strategy core.ErrorStrategy) EngineOption {
	return withErrorStrategy(strategy)
}

// WithErrorHandler 注入可插拔错误处理器。
func WithErrorHandler(h core.ErrorHandler) EngineOption {
	return withErrorHandler(h)
}

// WithFallbackFunc 配置兜底动作。
func WithFallbackFunc(fn func(ctx context.Context, chainID, ruleID string, data core.DataContext) error) EngineOption {
	return withFallbackFunc(fn)
}

// WithEvaluationMode 配置评估模式。
func WithEvaluationMode(mode core.EvaluationMode) EngineOption {
	return withEvaluationMode(mode)
}

// WithMetricsSink 配置标准化的指标导出器。
func WithMetricsSink(sink contract.MetricsSink) EngineOption {
	return withMetricsSink(sink)
}

// WithLogger 配置结构化日志。
func WithLogger(l contract.Logger) EngineOption {
	return withLogger(l)
}

// WithLimiter 配置限流器。
func WithLimiter(l contract.Limiter) EngineOption {
	return withLimiter(l)
}

// WithDebugManager 配置调试管理器（V12：改用接口）。
func WithDebugManager(dm contract.DebugManager) EngineOption {
	return withDebugManager(dm)
}

// WithStateStore 注入状态存储。
func WithStateStore(store core.StateStore) EngineOption {
	return withStateStore(store)
}

// WithEvalHook 注入评估钩子。
func WithEvalHook(hook EvalHook) EngineOption {
	return withEvalHook(hook)
}

// WithEvalTimeout 设置单次链评估的总超时。
func WithEvalTimeout(timeout time.Duration) EngineOption {
	return withEvalTimeout(timeout)
}

// WithResultPool 启用 Result 对象池。
func WithResultPool() EngineOption {
	return withResultPool()
}
