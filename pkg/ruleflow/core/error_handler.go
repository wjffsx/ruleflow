package core

// ─────────────────────────────────────────────
//  ErrorHandler — 可插拔错误处理器（V2 2.2 增强）
// ─────────────────────────────────────────────

// ErrorHandler 错误处理决策。
//
// 引擎在以下时机调用 HandleError 询问下一步动作：
//   - 条件评估抛错（condition evaluate failed）
//   - 动作执行抛错（action execute failed）
//   - panic 恢复（panic recovered）
//   - 依赖图错误（cyclic dependency 等加载阶段错误）
//
// 引擎本身已提供三个内建实现：
//   - ContinueOnError（默认）：跳过失败规则，继续评估
//   - AbortOnError：终止当前链评估
//   - RetryOnceError：重试一次仍失败则跳过
//
// 应用层可通过 WithErrorHandler 注入自定义 ErrorHandler 实现
// 复杂的 fallback、metrics 上报、远端告警等逻辑。
type ErrorHandler interface {
	// HandleError 处理错误并返回下一步动作。
	//
	// 入参：
	//   - ctx: 评估上下文
	//   - chainID: 所在链 ID
	//   - err: 标准化错误（*RuleFlowError）
	//
	// 返回：
	//   - ErrorAction: 引擎后续行为
	//   - error: 处理器自身错误（不影响控制流，仅记录日志）
	HandleError(ctx HandlerContext, err *RuleFlowError) (ErrorAction, error)
}

// ErrorAction 错误处理后引擎应采取的动作
type ErrorAction int

const (
	// ErrorActionContinue 继续评估下一条规则（默认）
	ErrorActionContinue ErrorAction = iota
	// ErrorActionAbort 终止当前链评估，立即返回
	ErrorActionAbort
	// ErrorActionRetry 对当前规则重试一次
	ErrorActionRetry
	// ErrorActionFallback 执行兜底（应用层可在 FallbackFunc 中定义兜底动作）
	ErrorActionFallback
)

// String 返回 ErrorAction 的可读表示
func (a ErrorAction) String() string {
	switch a {
	case ErrorActionContinue:
		return "continue"
	case ErrorActionAbort:
		return "abort"
	case ErrorActionRetry:
		return "retry"
	case ErrorActionFallback:
		return "fallback"
	default:
		return "unknown"
	}
}

// HandlerContext 错误处理时附带的上下文信息。
//
// 该结构体是只读的拷贝，处理器不应修改其内部字段。
type HandlerContext struct {
	// ChainID 触发错误的链 ID
	ChainID string
	// RuleID 触发错误的规则 ID（链级错误时为空）
	RuleID string
	// ActionID 触发错误的动作 ID（条件/链级错误时为空）
	ActionID string
	// ErrorType 错误类型枚举
	ErrorType ErrorType
	// Attempt 当前尝试次数（从 1 开始）
	Attempt int
}

// ─────────────────────────────────────────────
//  内建实现：Continue / Abort / Retry
// ─────────────────────────────────────────────

// continueOnError 默认处理器：忽略错误继续评估
type continueOnError struct{}

func (continueOnError) HandleError(_ HandlerContext, _ *RuleFlowError) (ErrorAction, error) {
	return ErrorActionContinue, nil
}

// abortOnError 终止评估
type abortOnError struct{}

func (abortOnError) HandleError(_ HandlerContext, _ *RuleFlowError) (ErrorAction, error) {
	return ErrorActionAbort, nil
}

// retryOnceError 重试一次，仍失败则跳过
type retryOnceError struct {
	maxAttempt int
}

func (r retryOnceError) HandleError(ctx HandlerContext, _ *RuleFlowError) (ErrorAction, error) {
	if ctx.Attempt < r.maxAttempt {
		return ErrorActionRetry, nil
	}
	return ErrorActionContinue, nil
}

// ContinueOnErrorHandler 返回"忽略错误继续评估"的处理器（默认行为）
func ContinueOnErrorHandler() ErrorHandler { return continueOnError{} }

// AbortOnErrorHandler 返回"遇错即停"的处理器
func AbortOnErrorHandler() ErrorHandler { return abortOnError{} }

// RetryOnceErrorHandler 返回"重试一次后跳过"的处理器
func RetryOnceErrorHandler() ErrorHandler { return retryOnceError{maxAttempt: 2} }

// ─────────────────────────────────────────────
//  链式 / 装饰器 ErrorHandler
// ─────────────────────────────────────────────

// ChainedErrorHandler 按顺序尝试多个处理器，返回第一个非 Continue 的动作。
// 用于"先重试，重试后上报 metrics，再继续"的组合场景。
type ChainedErrorHandler struct {
	Handlers []ErrorHandler
}

// HandleError 链式调用
func (c *ChainedErrorHandler) HandleError(ctx HandlerContext, err *RuleFlowError) (ErrorAction, error) {
	for _, h := range c.Handlers {
		action, herr := h.HandleError(ctx, err)
		if herr != nil {
			return action, herr
		}
		if action != ErrorActionContinue {
			return action, nil
		}
	}
	return ErrorActionContinue, nil
}

// MetricsErrorHandler 把错误计数委托给 MetricsSink 的装饰器。
// 应用层可包装任意 ErrorHandler，自动获得 metrics 上报。
type MetricsErrorHandler struct {
	Inner ErrorHandler
	// OnError 可选：每次错误触发（用于上报、告警等副作用）
	OnError func(ctx HandlerContext, err *RuleFlowError)
}

// HandleError 转发到 Inner
func (m *MetricsErrorHandler) HandleError(ctx HandlerContext, err *RuleFlowError) (ErrorAction, error) {
	if m.OnError != nil {
		m.OnError(ctx, err)
	}
	if m.Inner == nil {
		return ErrorActionContinue, nil
	}
	return m.Inner.HandleError(ctx, err)
}
