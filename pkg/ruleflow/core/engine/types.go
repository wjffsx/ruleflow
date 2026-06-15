// Package engine 提供轻量规则引擎（V7 §6.2 任务 6：934 行 engine.go 拆分）
//
// 文件结构（按职责划分）：
//   - engine.go:    顶层 API 透传（WithXxx 系列）
//   - types.go:     Engine 结构体 + EvalResult / RuleError / chainSnapshot 等内部类型
//   - options.go:   EngineOption 内部实现 + NewEngine + 公共方法
//   - load.go:      LoadChain / UnloadChain / depGraph 依赖图
//   - eval.go:      EvalChain 核心评估逻辑（含 panic recovery / 限流 / 背压）
//   - pool.go:      Result 对象池（AcquireResult / ReleaseResult）
//   - prewarm.go:   Prewarm / PrewarmAll
//   - health_shutdown.go: HealthCheck / Shutdown / 状态上报
//   - state.go:     WrapWithStateStore
//
// V12 迁移：移除 debug 包依赖，改用 contract.DebugManager 接口
package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/compiler"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  EvalResult — 评估结果
// ─────────────────────────────────────────────

// EngineOption 引擎配置选项（函数式）。
type EngineOption func(*Engine)

// RuleError 规则执行错误详情
type RuleError struct {
	RuleID   string `json:"rule_id"`
	ActionID string `json:"action_id,omitempty"`
	Err      error  `json:"-"`
	Message  string `json:"message"`
}

// EvalResult 规则链评估结果
type EvalResult struct {
	MatchedRules []*core.Rule     `json:"matched_rules"`
	Data         core.DataContext `json:"-"`
	Errors       []RuleError      `json:"errors,omitempty"`
	Dropped      bool             `json:"dropped"`
	Duration     time.Duration    `json:"duration"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// chainSnapshot COW 不可变快照
type chainSnapshot struct {
	chains map[string]*core.CompiledChain
	rules  map[string]*core.CompiledRule
}

// MaxMetadataSize Metadata map 最大容量（超过则丢弃，避免池化后内存泄漏）
const MaxMetadataSize = 32

// Reset 重置 EvalResult 到初始可复用状态。
func (r *EvalResult) Reset() {
	if r == nil {
		return
	}
	r.MatchedRules = r.MatchedRules[:0]
	r.Errors = r.Errors[:0]
	r.Data = nil
	r.Dropped = false
	if r.Metadata != nil {
		if len(r.Metadata) > MaxMetadataSize {
			r.Metadata = nil
		} else {
			for k := range r.Metadata {
				delete(r.Metadata, k)
			}
		}
	}
	r.Duration = 0
}

// ─────────────────────────────────────────────
//  EvalHook — 评估钩子
// ─────────────────────────────────────────────

// EvalHook 评估钩子，在 EvalChain 完成后回调。
// 服务层可实现此接口，将执行记录写入存储。
//
// V4.3 partial：接口保留在 engine 包是为了避免与 EvalResult 类型循环依赖
// （EvalHook.OnEval 签名直接引用 *EvalResult）。语义上属于"业务回调"而非"监控分层"。
type EvalHook interface {
	OnEval(ctx context.Context, chainID string, data core.DataContext, result *EvalResult)
}

// ─────────────────────────────────────────────
//  Engine — 引擎核心
// ─────────────────────────────────────────────

// Engine 轻量规则引擎
type Engine struct {
	snapshot atomic.Pointer[chainSnapshot] // COW 快照，零锁读取
	registry core.Registry                 // 实例级注册表（非全局）
	writeMu  sync.Mutex                    // 保护写入序列化（非读路径）

	// 可选组件（通过 EngineOption 注入，类型定义见 core/contract）
	tracer        contract.Tracer
	bpIndicator   contract.Indicator
	lossTracker   contract.Tracker
	actionTimeout time.Duration
	errorStrategy core.ErrorStrategy
	evalMode      core.EvaluationMode
	metricsSink   contract.MetricsSink
	logger        contract.Logger
	limiter       contract.Limiter
	errorHandler  core.ErrorHandler
	stateStore    core.StateStore
	evalHook      EvalHook
	evalTimeout   time.Duration
	fallbackFunc  func(ctx context.Context, chainID, ruleID string, data core.DataContext) error
	poolEnabled   bool

	// 优雅关闭 + 活跃评估
	shutdown   *shutdown
	activeEval int64

	// 依赖图（LoadChain/UnloadChain 使用）
	depGraph *compiler.DependencyGraph

	// 时间相关：闭包注入（V4.7）
	startTimeFn func() time.Time
	uptimeFn    func() time.Duration

	// 调试事件系统（V12：改用接口，零依赖）
	debugMgr contract.DebugManager
}
