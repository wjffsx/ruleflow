package core

import (
	"context"
	"fmt"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  DataContext — 数据点上下文（零分配设计）
// ─────────────────────────────────────────────

// DataContext 是规则引擎处理的数据上下文。
// 使用接口而非具体类型，允许接入不同数据模型。
//
// ⚠️ 并发安全声明：DataContext 实例不是并发安全的。
// 引擎保证每个 DataContext 实例在评估期间仅被单个 goroutine 访问。
// 如需并发访问，调用方需自行加锁。
type DataContext interface {
	// 基础标识
	DeviceID() string
	PointName() string
	PointType() string

	// 全限定名（格式: DeviceID/PointName）
	// 零分配设计：VPPTUDataContext 在 Acquire 时预计算并缓存，
	// 避免 fqn_prefix 条件评估时字符串拼接导致堆分配。
	FQN() string

	// 数据值
	Value() float64
	SetValue(v float64)

	// 质量与状态
	Quality() int
	SetQuality(q int)

	// 上下限
	UpperLimit() (float64, bool)
	LowerLimit() (float64, bool)
	LimitExceeded() bool
	SetLimitExceeded(v bool)

	// 元数据（零分配设计：GetTag 避免返回 map 引用）
	GetTag(key string) string
	SetTag(key, value string)

	// 路由目标（零分配设计：TargetAt/TargetCount 避免返回 slice 引用）
	TargetCount() int
	TargetAt(i int) string
	AddTarget(target string)

	// 标记
	Dropped() bool
	SetDropped(v bool)

	// 时间戳
	Timestamp() int64

	// 链路追踪（V2：使用 contract.SpanContext，零 otel 依赖）
	SpanContext() contract.SpanContext
	SetSpanContext(sc contract.SpanContext)

	// 前值（用于状态变化检测）
	PreviousValue() (float64, bool)
	SetPreviousValue(v float64)

	// 原始数据引用（扩展用）
	Raw() any
}

// ─────────────────────────────────────────────
//  Condition — 条件接口
// ─────────────────────────────────────────────

// Condition 条件评估接口。
// 预编译为函数闭包，Evaluate 无堆分配。
type Condition interface {
	// Evaluate 评估条件，返回是否匹配。
	// 热路径：不允许分配堆内存。
	Evaluate(ctx context.Context, data DataContext) bool

	// ID 条件唯一标识
	ID() string

	// Type 条件类型（用于序列化/可视化）
	Type() string

	// Description 人类可读描述
	Description() string
}

// ConditionFunc 函数式条件，允许直接用闭包创建条件
type ConditionFunc func(ctx context.Context, data DataContext) bool

func (f ConditionFunc) Evaluate(ctx context.Context, data DataContext) bool {
	return f(ctx, data)
}

func (f ConditionFunc) ID() string          { return "" }
func (f ConditionFunc) Type() string        { return "func" }
func (f ConditionFunc) Description() string { return "function condition" }

// ─────────────────────────────────────────────
//  Action — 动作接口
// ─────────────────────────────────────────────

// Action 动作执行接口。
// 预编译为函数闭包，Execute 尽量零分配。
type Action interface {
	// Execute 执行动作，修改 DataContext。
	// 返回 error 时进入错误处理链。
	Execute(ctx context.Context, data DataContext) error

	// ID 动作唯一标识
	ID() string

	// Type 动作类型
	Type() string

	// Description 人类可读描述
	Description() string
}

// ActionFunc 函数式动作
type ActionFunc func(ctx context.Context, data DataContext) error

func (f ActionFunc) Execute(ctx context.Context, data DataContext) error {
	return f(ctx, data)
}

func (f ActionFunc) ID() string          { return "" }
func (f ActionFunc) Type() string        { return "func" }
func (f ActionFunc) Description() string { return "function action" }

// ─────────────────────────────────────────────
//  Registry — 条件/动作注册表接口
// ─────────────────────────────────────────────

// Registry 条件/动作注册表接口（由 registry 包实现）
type Registry interface {
	CreateCondition(typeName, id string, config map[string]any) (Condition, error)
	CreateAction(typeName, id string, config map[string]any) (Action, error)
}

// DefaultRegistry 默认注册表（无内置注册，用于 Engine 初始化）
// 实际使用时应通过 WithRegistry() 注入 registry.NewEmptyRegistry()
type DefaultRegistry struct{}

func (r *DefaultRegistry) CreateCondition(typeName, id string, config map[string]any) (Condition, error) {
	return nil, fmt.Errorf("no registry configured: unknown condition type %s", typeName)
}

func (r *DefaultRegistry) CreateAction(typeName, id string, config map[string]any) (Action, error) {
	return nil, fmt.Errorf("no registry configured: unknown action type %s", typeName)
}

// ─────────────────────────────────────────────
//  StateStore — 有状态条件的状态存储
// ─────────────────────────────────────────────

// StateStore 有状态条件的状态存储接口。
// 可选注入到引擎，不注入时有状态条件（duration/trend/periodic）始终返回 false。
type StateStore interface {
	// Get 获取指定 key 的状态
	Get(key string) (any, bool)
	// Set 设置指定 key 的状态
	Set(key string, value any)
	// Delete 删除指定 key 的状态
	Delete(key string)
}

// StatefulDataContext 扩展 DataContext 以支持状态存储
type StatefulDataContext interface {
	DataContext
	StateStore() StateStore
}

// ─────────────────────────────────────────────
//  Prewarmable — 预热接口
// ─────────────────────────────────────────────

// Prewarmable 标识可被预热的对象（条件/动作/规则链等）。
//
// 引擎的 Prewarm 流程会：
//  1. 对链中每条规则调用其条件/动作的 Prewarm
//  2. 应用层可在自定义 Condition/Action 上实现该接口
//  3. 例如：连接池预热、缓存预热、模型加载
type Prewarmable interface {
	// Prewarm 预热当前对象。ctx 用于取消。
	Prewarm(ctx context.Context) error
}
