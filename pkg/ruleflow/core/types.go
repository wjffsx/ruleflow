package core

import (
	"context"
	"errors"
	"time"
)

// ─────────────────────────────────────────────
//  错误处理策略与评估模式
// ─────────────────────────────────────────────

// ErrorStrategy 规则执行失败时的处理策略
type ErrorStrategy int

const (
	// ErrorStrategyContinue 跳过失败规则，继续评估后续规则（默认）
	ErrorStrategyContinue ErrorStrategy = iota
	// ErrorStrategyAbort 终止当前链评估，立即返回错误
	ErrorStrategyAbort
)

// EvaluationMode 规则链评估模式
type EvaluationMode int

const (
	// EvalModeAll 评估所有匹配规则（默认）
	EvalModeAll EvaluationMode = iota
	// EvalModeFirst 首次匹配后终止
	EvalModeFirst
)

// ─────────────────────────────────────────────
//  LogicalOperator — 逻辑运算符
// ─────────────────────────────────────────────

// LogicalOperator 逻辑运算符
type LogicalOperator int

const (
	OpAnd LogicalOperator = iota
	OpOr
	OpNot
)

// ─────────────────────────────────────────────
//  ConditionNode — 条件树节点
// ─────────────────────────────────────────────

// ConditionNode 条件树节点
type ConditionNode struct {
	ID       string           `json:"id"`
	Operator LogicalOperator  `json:"operator"`
	Children []*ConditionNode `json:"children,omitempty"`
	Leaf     Condition        `json:"-"` // 叶节点条件（不序列化）

	// 序列化字段（用于配置持久化和可视化）
	LeafType   string         `json:"leaf_type,omitempty"`
	LeafConfig map[string]any `json:"leaf_config,omitempty"`
}

// Evaluate 递归评估条件树
func (n *ConditionNode) Evaluate(ctx context.Context, data DataContext) bool {
	if n.Leaf != nil {
		return n.Leaf.Evaluate(ctx, data)
	}

	switch n.Operator {
	case OpAnd:
		for _, child := range n.Children {
			if !child.Evaluate(ctx, data) {
				return false
			}
		}
		return true
	case OpOr:
		for _, child := range n.Children {
			if child.Evaluate(ctx, data) {
				return true
			}
		}
		return false
	case OpNot:
		if len(n.Children) > 0 {
			return !n.Children[0].Evaluate(ctx, data)
		}
		return true
	default:
		return false
	}
}

// ─────────────────────────────────────────────
//  ActionChain — 动作链
// ─────────────────────────────────────────────

// ErrDropData 特殊错误：标记数据点应被丢弃
var ErrDropData = errors.New("drop data point")

// ActionChain 有序动作链，按顺序执行
type ActionChain struct {
	Actions []Action `json:"actions"`
}

// Execute 按序执行所有动作
// 任何 Action 返回 ErrDropData 时立即终止链并标记丢弃
func (c *ActionChain) Execute(ctx context.Context, data DataContext) error {
	for _, action := range c.Actions {
		if err := action.Execute(ctx, data); err != nil {
			if err == ErrDropData {
				data.SetDropped(true)
				return nil
			}
			return err
		}
	}
	return nil
}

// ─────────────────────────────────────────────
//  Rule — 规则定义
// ─────────────────────────────────────────────

// Rule 规则 = 条件树 + 动作链
type Rule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    int    `json:"priority"` // 优先级，值越小越先执行
	// ★ Phase 3：权重
	// 在 Priority 相等时按 Weight 排序（值越小越先执行；默认 0）
	Weight int `json:"weight,omitempty"`
	// ★ Phase 3：编译期序号（由 compiler 填入；应用层通常无需设置）
	SequenceNo int  `json:"sequence_no,omitempty"`
	Enabled    bool `json:"enabled"`

	// ★ 输入绑定机制（Phase 1 新增）
	InputBindings []string `json:"input_bindings,omitempty"` // 引用的输入点名列表
	InputMode     string   `json:"input_mode,omitempty"`     // "single" | "multi"

	Condition *ConditionNode `json:"condition"` // 条件树
	Actions   *ActionChain   `json:"actions"`   // 动作链
	Targets   []string       `json:"targets"`   // 匹配后的路由目标

	// 元信息
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─────────────────────────────────────────────
//  RuleChain — 规则链
// ─────────────────────────────────────────────

// RuleChain 规则链：多条规则的有序集合
type RuleChain struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Root        bool   `json:"root"` // 是否为根链
	Version     int    `json:"version"`
	Status      string `json:"status"` // draft / deployed / archived

	// ★ 类型契约机制（Phase 1 新增）
	PipelineType string           `json:"pipeline_type,omitempty"` // "analog" | "digital" | "meter" | ""
	Inputs       []RuleChainInput `json:"inputs,omitempty"`        // 输入数据点声明

	Rules   []*Rule           `json:"rules"`
	Outputs []RuleChainOutput `json:"outputs,omitempty"`

	// ★ 依赖关系（MVP 新增）：当前链引用的其他链 ID 列表
	// 用于依赖图检测和循环引用防护
	Refs []string `json:"refs,omitempty"`
}

// RuleChainInput 规则链输入声明（Phase 1 新增）
type RuleChainInput struct {
	PointName   string `json:"point_name"`
	DisplayName string `json:"display_name"`
	PointType   string `json:"point_type"` // analog/digital/meter
	DataType    string `json:"data_type"`  // double/int/bool/string
	Unit        string `json:"unit"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

// RuleChainOutput 规则链输出声明
type RuleChainOutput struct {
	PointName   string   `json:"point_name"`
	DisplayName string   `json:"display_name"`
	PointType   string   `json:"point_type"`
	DataType    string   `json:"data_type"`
	Unit        string   `json:"unit"`
	Group       string   `json:"group"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
	InputPoints []string `json:"input_points"`
}

// ─────────────────────────────────────────────
//  CompiledChain / CompiledRule — 编译后的规则链
// ─────────────────────────────────────────────

// CompiledChain 编译后的规则链
type CompiledChain struct {
	ChainID       string
	Version       int
	SortedRules   []*CompiledRule // 按 Priority 排序
	FastRules     []*CompiledRule // FastPath 规则子集（< 200ns）
	SlowRules     []*CompiledRule // SlowPath 规则子集（< 10μs）
	CompiledAt    time.Time       // 编译时间
	EvalMode      EvaluationMode  // 评估模式
	ErrorStrategy ErrorStrategy   // 错误策略
}

// CompiledRule 编译后的规则
type CompiledRule struct {
	Rule         *Rule
	IsFast       bool
	EvaluateFunc func(ctx context.Context, data DataContext) bool  // 预编译条件
	ExecuteFunc  func(ctx context.Context, data DataContext) error // 预编译动作链
	// ★ Phase 3：聚合后的 prewarm 函数（nil 表示无可预热对象）
	PrewarmFunc func(ctx context.Context) error
}
