package ext

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  ExprSwitch action — 表达式分支动作
// ─────────────────────────────────────────────

// ExprSwitchAction 根据表达式结果分流
type ExprSwitchAction struct {
	IDValue      string
	Expression   string
	compiledProg *vm.Program
}

// exprEnvPool 复用 expr 环境 map
var exprEnvPool = sync.Pool{
	New: func() any { return make(map[string]any, 10) },
}

var _ core.Action = (*ExprSwitchAction)(nil)

func NewExprSwitchAction(id, expression string) *ExprSwitchAction {
	a := &ExprSwitchAction{
		IDValue:    id,
		Expression: expression,
	}
	if expression != "" {
		env := map[string]any{
			"value":     float64(0),
			"device_id": "",
			"point":     "",
		}
		prog, err := expr.Compile(expression, expr.Env(env))
		if err == nil {
			a.compiledProg = prog
		}
	}
	return a
}

func (a *ExprSwitchAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Expression == "" || a.compiledProg == nil {
		return fmt.Errorf("expr_switch: expression is empty")
	}

	env := exprEnvPool.Get().(map[string]any)
	defer func() {
		for k := range env {
			delete(env, k)
		}
		exprEnvPool.Put(env)
	}()

	env["value"] = data.Value()
	env["device_id"] = data.DeviceID()
	env["point"] = data.PointName()
	env["quality"] = data.Quality()
	env["ts"] = data.Timestamp()
	env["exceeded"] = data.LimitExceeded()

	out, err := expr.Run(a.compiledProg, env)
	if err != nil {
		// 表达式执行失败，走 false 分支
		data.SetTag("_switch_result", "false")
		return nil
	}

	var result bool
	switch v := out.(type) {
	case bool:
		result = v
	case string:
		result, err = strconv.ParseBool(v)
		if err != nil {
			result = false
		}
	default:
		result = false
	}

	if result {
		data.SetTag("_switch_result", "true")
	} else {
		data.SetTag("_switch_result", "false")
	}

	return nil
}

func (a *ExprSwitchAction) ID() string          { return a.IDValue }
func (a *ExprSwitchAction) Type() string        { return "expr_switch" }
func (a *ExprSwitchAction) Description() string { return "expression switch" }
