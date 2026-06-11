package ext

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  CalcNodeAction — 计算节点动作（expr-lang）
// ─────────────────────────────────────────────

var (
	caretPowRe = regexp.MustCompile(`([a-zA-Z_]\w*)\s*\^\s*(\d+(?:\.\d+)?)`)
	calcFuncs  = map[string]any{
		"sqrt": math.Sqrt,
		"abs":  math.Abs,
		"pow":  math.Pow,
		"min":  math.Min,
		"max":  math.Max,
	}
)

func preprocess(formula string) string {
	return caretPowRe.ReplaceAllString(formula, "pow($1, $2)")
}

// CalcNodeAction 使用 expr-lang 执行公式计算
type CalcNodeAction struct {
	IDValue          string
	Formula          string
	Inputs           []string
	Output           string
	processedFormula string
	compiledExpr     *vm.Program
}

var _ core.Action = (*CalcNodeAction)(nil)

func NewCalcNodeAction(id, formula string, inputs []string, output string) *CalcNodeAction {
	a := &CalcNodeAction{
		IDValue: id,
		Formula: formula,
		Inputs:  inputs,
		Output:  output,
	}
	a.processedFormula = preprocess(formula)

	if formula != "" {
		baseEnv := make(map[string]any, 8+len(inputs))
		// 始终注入基础变量
		baseEnv["value"] = float64(0)
		baseEnv["device_id"] = ""
		baseEnv["point_name"] = ""
		baseEnv["quality"] = float64(0)
		for _, input := range inputs {
			if _, ok := baseEnv[input]; !ok {
				baseEnv[input] = float64(0)
			}
		}
		for k, v := range calcFuncs {
			baseEnv[k] = v
		}
		prog, err := expr.Compile(a.processedFormula, expr.Env(baseEnv), expr.AsFloat64())
		if err == nil {
			a.compiledExpr = prog
		}
	}
	return a
}

func (a *CalcNodeAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Formula == "" {
		return fmt.Errorf("calc action: formula is empty")
	}

	// 每次新建 env map，避免对象池复用导致的诡异 nil 问题
	env := make(map[string]any, 12)

	// 始终注入基础变量，确保公式可以引用 value/quality 等
	env["value"] = data.Value()
	env["device_id"] = data.DeviceID()
	env["point_name"] = data.PointName()
	env["quality"] = float64(data.Quality())

	for k, v := range calcFuncs {
		env[k] = v
	}

	// 从 DataContext 读取输入值
	for _, input := range a.Inputs {
		var val float64

		// 尝试从 Tag 读取（metadata 视角）
		tagVal := data.GetTag(input)
		if tagVal != "" {
			if f, err := strconv.ParseFloat(tagVal, 64); err == nil {
				val = f
				env[input] = val
				continue
			}
		}

		// 使用 input 作为 deviceID 的补充
		env[input] = data.Value()
	}

	result, err := expr.Run(a.compiledExpr, env)
	if err != nil {
		return fmt.Errorf("calc action: formula evaluation failed: %w", err)
	}

	r, ok := result.(float64)
	if !ok {
		return fmt.Errorf("calc action: formula result is not float64")
	}

	// 设置结果到 DataContext
	data.SetValue(r)
	data.SetTag("calc_result", strconv.FormatFloat(r, 'f', -1, 64))
	data.SetTag("calc_output", a.Output)

	return nil
}

func (a *CalcNodeAction) ID() string          { return a.IDValue }
func (a *CalcNodeAction) Type() string        { return "calc_node" }
func (a *CalcNodeAction) Description() string { return "expr-lang calculation" }
