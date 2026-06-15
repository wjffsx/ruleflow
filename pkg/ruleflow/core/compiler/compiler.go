package compiler

import (
	"context"
	"fmt"
	"sort"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  规则编译器
// ─────────────────────────────────────────────

// MaxConditionDepth 条件树最大深度（防止栈溢出）
const MaxConditionDepth = 32

// FastPathActionTypes FastPath 兼容的动作类型（< 200ns）
var FastPathActionTypes = map[string]bool{
	"transform": true, "rename": true, "tag": true,
	"drop": true, "route": true, "limit_check": true, "quality_mark": true,
}

// fastPathConditionTypes FastPath 兼容的条件类型（< 100ns，或零分配且 < 500ns）
// fqn_prefix 通过 DataContext.FQN() 预计算缓存实现零分配，约 427ns，归为 FastPath
// value_in 通过预编译 map[float64]struct{} 实现 O(1) 查找，归为 FastPath
var fastPathConditionTypes = map[string]bool{
	"device_type": true, "point_name": true, "point_name_pattern": true,
	"value_range": true, "quality": true, "limit_exceeded": true,
	"device_id": true, "fqn_prefix": true, "value_in": true,
}

// FastPathConditionTypes 保留公开只读访问（向后兼容）
var FastPathConditionTypes = fastPathConditionTypes

// IsFastPathCondition 判断条件类型是否属于 FastPath
func IsFastPathCondition(typeName string) bool {
	return fastPathConditionTypes[typeName]
}

// RegisterFastPathCondition 注册新的 FastPath 条件类型
func RegisterFastPathCondition(typeName string) {
	fastPathConditionTypes[typeName] = true
}

// UnregisterFastPathCondition 取消注册 FastPath 条件类型
func UnregisterFastPathCondition(typeName string) {
	delete(fastPathConditionTypes, typeName)
}

// MaxActionChainLength 动作链最大长度（防止死循环）
const MaxActionChainLength = 32

// CompileChain 将 RuleChain 编译为 CompiledChain（导出版本，供 benchmark/外部调用）
func CompileChain(chain *core.RuleChain, registry core.Registry) (*core.CompiledChain, error) {
	return compileChain(chain, registry)
}

// compileChain 将 RuleChain 编译为 CompiledChain
func compileChain(chain *core.RuleChain, registry core.Registry) (*core.CompiledChain, error) {
	compiledRules := make([]*core.CompiledRule, 0, len(chain.Rules))
	for _, rule := range chain.Rules {
		if !rule.Enabled {
			continue
		}
		cr, err := compileRule(rule)
		if err != nil {
			return nil, fmt.Errorf("compile rule %s: %w", rule.ID, err)
		}
		compiledRules = append(compiledRules, cr)
	}

	// ★ Phase 3：稳定排序
	// 1) Priority 升序（小者优先）
	// 2) Weight   升序（同级内更轻者优先）
	// 3) ID       字典序（保证稳定性 + 可重现）
	sort.SliceStable(compiledRules, func(i, j int) bool {
		a, b := compiledRules[i].Rule, compiledRules[j].Rule
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if a.Weight != b.Weight {
			return a.Weight < b.Weight
		}
		return a.ID < b.ID
	})

	// 回填 SequenceNo（应用层在日志/指标中可引用）
	for i, cr := range compiledRules {
		cr.Rule.SequenceNo = i
	}

	// 分类 Fast/Slow 规则
	var fastRules, slowRules []*core.CompiledRule
	for _, cr := range compiledRules {
		if cr.IsFast {
			fastRules = append(fastRules, cr)
		} else {
			slowRules = append(slowRules, cr)
		}
	}

	return &core.CompiledChain{
		ChainID:     chain.ID,
		Version:     chain.Version,
		SortedRules: compiledRules,
		FastRules:   fastRules,
		SlowRules:   slowRules,
	}, nil
}

// CompileRule 将 Rule 编译为 CompiledRule（导出版本）
func CompileRule(rule *core.Rule) (*core.CompiledRule, error) {
	return compileRule(rule)
}

// compileRule 将 Rule 编译为 CompiledRule
func compileRule(rule *core.Rule) (*core.CompiledRule, error) {
	// 校验条件树深度
	if rule.Condition != nil {
		if err := validateConditionDepth(rule.Condition, 0, MaxConditionDepth); err != nil {
			return nil, fmt.Errorf("condition tree too deep: %w", err)
		}
	}

	evalFunc, err := compileConditionTree(rule.Condition)
	if err != nil {
		return nil, fmt.Errorf("compile condition: %w", err)
	}

	execFunc, isFast, err := compileActionChain(rule.Actions)
	if err != nil {
		return nil, fmt.Errorf("compile actions: %w", err)
	}

	// FastPath 判定：条件 + 动作全部兼容
	conditionFast := isConditionFast(rule.Condition)

	// ★ Phase 3：聚合 prewarm 钩子
	prewarmFunc := buildPrewarmFunc(rule.Condition, rule.Actions)

	return &core.CompiledRule{
		Rule:         rule,
		IsFast:       conditionFast && isFast,
		EvaluateFunc: evalFunc,
		ExecuteFunc:  execFunc,
		PrewarmFunc:  prewarmFunc,
	}, nil
}

// buildPrewarmFunc 聚合条件树和动作链中的所有 Prewarmable，
// 返回一个组合的 prewarm 函数；若无可预热对象则返回 nil。
func buildPrewarmFunc(cond *core.ConditionNode, actions *core.ActionChain) func(ctx context.Context) error {
	var hooks []func(ctx context.Context) error

	// 遍历条件树
	var walk func(n *core.ConditionNode)
	walk = func(n *core.ConditionNode) {
		if n == nil {
			return
		}
		if n.Leaf != nil {
			if p, ok := n.Leaf.(core.Prewarmable); ok {
				hooks = append(hooks, p.Prewarm)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(cond)

	// 遍历动作链
	if actions != nil {
		for _, a := range actions.Actions {
			if p, ok := a.(core.Prewarmable); ok {
				hooks = append(hooks, p.Prewarm)
			}
		}
	}

	if len(hooks) == 0 {
		return nil
	}

	return func(ctx context.Context) error {
		for _, h := range hooks {
			if err := h(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

// validateConditionDepth 校验条件树深度防止栈溢出
func validateConditionDepth(node *core.ConditionNode, current, max int) error {
	if current > max {
		return fmt.Errorf("depth %d exceeds max %d", current, max)
	}
	for _, child := range node.Children {
		if err := validateConditionDepth(child, current+1, max); err != nil {
			return err
		}
	}
	return nil
}

// isConditionFast 判断条件树是否全部使用 FastPath 兼容条件
func isConditionFast(node *core.ConditionNode) bool {
	if node == nil {
		return true
	}
	if node.Leaf != nil {
		return FastPathConditionTypes[node.Leaf.Type()]
	}
	for _, child := range node.Children {
		if !isConditionFast(child) {
			return false
		}
	}
	return true
}

// CompileConditionTree 将条件树递归编译为单一闭包（导出版本）
func CompileConditionTree(node *core.ConditionNode) (func(ctx context.Context, data core.DataContext) bool, error) {
	return compileConditionTree(node)
}

// compileConditionTree 将条件树递归编译为单一闭包
// 编译结果是一个 func(ctx, data) bool，运行时零分配
func compileConditionTree(node *core.ConditionNode) (func(ctx context.Context, data core.DataContext) bool, error) {
	if node == nil {
		return func(_ context.Context, _ core.DataContext) bool { return true }, nil
	}

	// 叶节点：直接调用 Condition.Evaluate
	if node.Leaf != nil {
		leaf := node.Leaf
		return func(ctx context.Context, data core.DataContext) bool {
			return leaf.Evaluate(ctx, data)
		}, nil
	}

	// 内部节点：递归编译子节点
	childFuncs := make([]func(ctx context.Context, data core.DataContext) bool, len(node.Children))
	for i, child := range node.Children {
		cf, err := compileConditionTree(child)
		if err != nil {
			return nil, err
		}
		childFuncs[i] = cf
	}

	switch node.Operator {
	case core.OpAnd:
		return func(ctx context.Context, data core.DataContext) bool {
			for _, f := range childFuncs {
				if !f(ctx, data) {
					return false // 短路求值
				}
			}
			return true
		}, nil
	case core.OpOr:
		return func(ctx context.Context, data core.DataContext) bool {
			for _, f := range childFuncs {
				if f(ctx, data) {
					return true // 短路求值
				}
			}
			return false
		}, nil
	case core.OpNot:
		if len(childFuncs) > 0 {
			f := childFuncs[0]
			return func(ctx context.Context, data core.DataContext) bool {
				return !f(ctx, data)
			}, nil
		}
		return func(_ context.Context, _ core.DataContext) bool { return false }, nil
	default:
		return nil, fmt.Errorf("unknown operator: %d", node.Operator)
	}
}

// CompileActionChain 将动作链编译为单一闭包（导出版本）
func CompileActionChain(chain *core.ActionChain) (func(ctx context.Context, data core.DataContext) error, bool, error) {
	return compileActionChain(chain)
}

// compileActionChain 将动作链编译为单一闭包
// 返回闭包 + 是否全部 FastPath 兼容
func compileActionChain(chain *core.ActionChain) (func(ctx context.Context, data core.DataContext) error, bool, error) {
	if chain == nil || len(chain.Actions) == 0 {
		return nil, true, nil
	}

	if len(chain.Actions) > MaxActionChainLength {
		return nil, false, fmt.Errorf("action chain too long: %d > %d", len(chain.Actions), MaxActionChainLength)
	}

	actions := chain.Actions
	allFast := true
	for _, a := range actions {
		if !FastPathActionTypes[a.Type()] {
			allFast = false
		}
	}

	return func(ctx context.Context, data core.DataContext) error {
		for _, action := range actions {
			if err := action.Execute(ctx, data); err != nil {
				if err == core.ErrDropData {
					data.SetDropped(true)
					return nil
				}
				return err
			}
		}
		return nil
	}, allFast, nil
}
