package engine

import (
	"context"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/compiler"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  EngineOption 内部实现（小写命名，对外 API 由 engine.go 透传）
// ─────────────────────────────────────────────

// withTracer 配置追踪器。
func withTracer(tp contract.TracerProvider) EngineOption {
	return func(e *Engine) {
		if tp != nil {
			e.tracer = tp.Tracer("ruleflow")
		}
	}
}

// withBackpressureIndicator 配置背压指示器。
func withBackpressureIndicator(bp contract.Indicator) EngineOption {
	return func(e *Engine) { e.bpIndicator = bp }
}

// withDataLossTracker 配置数据丢失追踪器。
func withDataLossTracker(dlt contract.Tracker) EngineOption {
	return func(e *Engine) { e.lossTracker = dlt }
}

// withRegistry 配置注册表
func withRegistry(r core.Registry) EngineOption {
	return func(e *Engine) { e.registry = r }
}

// withActionTimeout 配置单次 action 执行的超时。
func withActionTimeout(timeout time.Duration) EngineOption {
	return func(e *Engine) { e.actionTimeout = timeout }
}

// withErrorStrategy 配置错误策略
func withErrorStrategy(strategy core.ErrorStrategy) EngineOption {
	return func(e *Engine) { e.errorStrategy = strategy }
}

// withErrorHandler 注入可插拔错误处理器。
func withErrorHandler(h core.ErrorHandler) EngineOption {
	return func(e *Engine) {
		if h != nil {
			e.errorHandler = h
		}
	}
}

// withFallbackFunc 配置兜底动作。
func withFallbackFunc(fn func(ctx context.Context, chainID, ruleID string, data core.DataContext) error) EngineOption {
	return func(e *Engine) {
		if fn != nil {
			e.fallbackFunc = fn
		}
	}
}

// withEvaluationMode 配置评估模式
func withEvaluationMode(mode core.EvaluationMode) EngineOption {
	return func(e *Engine) { e.evalMode = mode }
}

// withMetricsSink 配置标准化的指标导出器。
func withMetricsSink(sink contract.MetricsSink) EngineOption {
	return func(e *Engine) {
		if sink != nil {
			e.metricsSink = sink
		}
	}
}

// withLogger 配置结构化日志。
func withLogger(l contract.Logger) EngineOption {
	return func(e *Engine) {
		if l != nil {
			e.logger = l
		}
	}
}

// withLimiter 配置限流器。
func withLimiter(l contract.Limiter) EngineOption {
	return func(e *Engine) {
		if l != nil {
			e.limiter = l
		}
	}
}

// withDebugManager 配置调试管理器（V12：改用接口）。
func withDebugManager(dm contract.DebugManager) EngineOption {
	return func(e *Engine) {
		if dm != nil {
			e.debugMgr = dm
		}
	}
}

// withStateStore 注入状态存储。
func withStateStore(store core.StateStore) EngineOption {
	return func(e *Engine) { e.stateStore = store }
}

// withEvalHook 注入评估钩子。
func withEvalHook(hook EvalHook) EngineOption {
	return func(e *Engine) { e.evalHook = hook }
}

// withEvalTimeout 设置单次链评估的总超时。
func withEvalTimeout(timeout time.Duration) EngineOption {
	return func(e *Engine) { e.evalTimeout = timeout }
}

// withResultPool 启用 Result 对象池（V2 3.4 内存池化）。
func withResultPool() EngineOption {
	return func(e *Engine) { e.poolEnabled = true }
}

// NewEngine 创建引擎实例
func NewEngine(opts ...EngineOption) *Engine {
	now := time.Now()
	e := &Engine{
		registry:     &core.DefaultRegistry{},
		shutdown:     newShutdown(),
		depGraph:     compiler.NewDependencyGraph(),
		startTimeFn:  func() time.Time { return now },
		uptimeFn:     func() time.Duration { return time.Since(now) },
		metricsSink:  contract.NoopSink(),
		logger:       contract.NoopLogger(),
		limiter:      contract.NoopLimiter(),
		errorHandler: core.ContinueOnErrorHandler(),
		poolEnabled:  false,
	}
	e.snapshot.Store(&chainSnapshot{
		chains: make(map[string]*core.CompiledChain),
		rules:  make(map[string]*core.CompiledRule),
	})
	for _, opt := range opts {
		opt(e)
	}
	if e.metricsSink == nil {
		e.metricsSink = contract.NoopSink()
	}
	if e.logger == nil {
		e.logger = contract.NoopLogger()
	}
	if e.limiter == nil {
		e.limiter = contract.NoopLimiter()
	}
	e.metricsSink.SetLoadedChains(len(e.getChainIDs()))
	return e
}

// ─────────────────────────────────────────────
//  公共方法（透传至 load.go / eval.go 的小写实现）
// ─────────────────────────────────────────────

// LoadChain 加载并编译规则链。
func (e *Engine) LoadChain(chain *core.RuleChain) error {
	return e.loadChain(chain)
}

// UnloadChain 卸载规则链。
func (e *Engine) UnloadChain(chainID string) {
	e.unloadChain(chainID)
}

// GetChainIDs 返回已加载的链 ID 列表。
func (e *Engine) GetChainIDs() []string {
	return e.getChainIDs()
}

// GetDebugManager 返回调试管理器（V12：返回接口类型）。
func (e *Engine) GetDebugManager() contract.DebugManager {
	return e.debugMgr
}

// EvalChain 评估规则链（核心热路径）。
func (e *Engine) EvalChain(ctx context.Context, chainID string, data core.DataContext) (result *EvalResult, err error) {
	return e.evalChain(ctx, chainID, data)
}

// EvalChainBatch 批量评估。
func (e *Engine) EvalChainBatch(ctx context.Context, chainID string, dataList []core.DataContext) ([]*EvalResult, error) {
	if e.poolEnabled {
		return e.evalChainBatchPooled(ctx, chainID, dataList)
	}
	return e.evalChainBatchPlain(ctx, chainID, dataList)
}
