package ext

import (
	"context"
	"strconv"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  ExprFilterCondition — 表达式条件（exprFilter 迁移）
// ─────────────────────────────────────────────

// ExprFilterCondition 使用 expr-lang 表达式进行布尔评估的条件叶节点
type ExprFilterCondition struct {
	IDValue      string
	Expression   string
	compiledProg *vm.Program
}

var _ core.Condition = (*ExprFilterCondition)(nil)

func NewExprFilterCondition(id, expression string) *ExprFilterCondition {
	c := &ExprFilterCondition{
		IDValue:    id,
		Expression: expression,
	}
	if expression != "" {
		// 预编译表达式
		env := map[string]any{
			"value":     float64(0),
			"device_id": "",
			"point":     "",
			"quality":   0,
			"tag":       map[string]string{},
		}
		prog, err := expr.Compile(expression, expr.Env(env))
		if err == nil {
			c.compiledProg = prog
		}
	}
	return c
}

func (c *ExprFilterCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	if c.Expression == "" || c.compiledProg == nil {
		return true
	}

	env := map[string]any{
		"value":     data.Value(),
		"device_id": data.DeviceID(),
		"point":     data.PointName(),
		"quality":   data.Quality(),
		"pointType": data.PointType(),
		"ts":        data.Timestamp(),
		"exceeded":  data.LimitExceeded(),
		"dropped":   data.Dropped(),
	}

	out, err := expr.Run(c.compiledProg, env)
	if err != nil {
		return false
	}

	result, ok := out.(bool)
	if !ok {
		// 尝试 string → bool
		if s, ok := out.(string); ok {
			if b, err := strconv.ParseBool(s); err == nil {
				return b
			}
		}
		return false
	}
	return result
}

func (c *ExprFilterCondition) ID() string          { return c.IDValue }
func (c *ExprFilterCondition) Type() string        { return "expr_filter" }
func (c *ExprFilterCondition) Description() string { return "expression filter condition" }
