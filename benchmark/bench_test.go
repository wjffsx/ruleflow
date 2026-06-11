package benchmark

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/action"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/compiler"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"
)

// ─────────────────────────────────────────────
//  零分配 DataContext（sync.Pool 复用）
// ─────────────────────────────────────────────

type benchDataPoint struct {
	deviceID      string
	pointName     string
	pointType     string
	fqn           string // 预计算缓存，模拟 VPPTUDataContext
	value         float64
	quality       int
	limitExceeded bool
	dropped       bool
	targets       []string
	tags          map[string]string
	upperLimit    float64
	hasUpper      bool
	lowerLimit    float64
	hasLower      bool
}

func (d *benchDataPoint) DeviceID() string                      { return d.deviceID }
func (d *benchDataPoint) PointName() string                     { return d.pointName }
func (d *benchDataPoint) PointType() string                     { return d.pointType }
func (d *benchDataPoint) FQN() string                           { return d.fqn }
func (d *benchDataPoint) Value() float64                        { return d.value }
func (d *benchDataPoint) SetValue(v float64)                    { d.value = v }
func (d *benchDataPoint) Quality() int                          { return d.quality }
func (d *benchDataPoint) SetQuality(q int)                      { d.quality = q }
func (d *benchDataPoint) UpperLimit() (float64, bool)           { return d.upperLimit, d.hasUpper }
func (d *benchDataPoint) LowerLimit() (float64, bool)           { return d.lowerLimit, d.hasLower }
func (d *benchDataPoint) LimitExceeded() bool                   { return d.limitExceeded }
func (d *benchDataPoint) SetLimitExceeded(v bool)               { d.limitExceeded = v }
func (d *benchDataPoint) GetTag(key string) string              { return d.tags[key] }
func (d *benchDataPoint) SetTag(key, value string)              { d.tags[key] = value }
func (d *benchDataPoint) TargetCount() int                      { return len(d.targets) }
func (d *benchDataPoint) TargetAt(i int) string                 { return d.targets[i] }
func (d *benchDataPoint) AddTarget(target string)               { d.targets = append(d.targets, target) }
func (d *benchDataPoint) Dropped() bool                         { return d.dropped }
func (d *benchDataPoint) SetDropped(v bool)                     { d.dropped = v }
func (d *benchDataPoint) Timestamp() int64                      { return 0 }
func (d *benchDataPoint) SpanContext() contract.SpanContext     { return contract.SpanContext{} }
func (d *benchDataPoint) SetSpanContext(_ contract.SpanContext) {}
func (d *benchDataPoint) Raw() any                              { return d }
func (d *benchDataPoint) PreviousValue() (float64, bool)        { return 0, false }
func (d *benchDataPoint) SetPreviousValue(v float64)            {}

func (d *benchDataPoint) reset() {
	d.deviceID = "device_001"
	d.pointName = "voltage"
	d.fqn = "device_001/voltage"
	d.pointType = "analog"
	d.value = 220.5
	d.quality = 192
	d.limitExceeded = false
	d.dropped = false
	d.targets = d.targets[:0]
	for k := range d.tags {
		delete(d.tags, k)
	}
	d.upperLimit = 250.0
	d.hasUpper = true
	d.lowerLimit = 200.0
	d.hasLower = true
}

func newBenchData() *benchDataPoint {
	return &benchDataPoint{
		deviceID:   "device_001",
		pointName:  "voltage",
		fqn:        "device_001/voltage",
		pointType:  "analog",
		value:      220.5,
		quality:    192,
		upperLimit: 250.0,
		hasUpper:   true,
		lowerLimit: 200.0,
		hasLower:   true,
		tags:       make(map[string]string),
	}
}

// ─────────────────────────────────────────────
//  8 种内置条件 Benchmark — 纯评估（零分配目标 < 100ns/op）
//  数据点在循环外预分配，仅测量条件评估本身
// ─────────────────────────────────────────────

func BenchmarkCondition_DeviceType(b *testing.B) {
	cond := condition.NewDeviceTypeCondition("c1", []string{"analog", "digital"})
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_PointName(b *testing.B) {
	cond := condition.NewPointNameCondition("c1", []string{"voltage", "current", "power"})
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_PointNamePattern(b *testing.B) {
	cond, _ := condition.NewPointNamePatternCondition("c1", `^volt(age)?_[0-9]+$`)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_ValueRange(b *testing.B) {
	min, max := 200.0, 250.0
	cond := condition.NewValueRangeCondition("c1", &min, &max)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_Quality(b *testing.B) {
	cond := condition.NewQualityCondition("c1", 128)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_LimitExceeded(b *testing.B) {
	cond := condition.NewLimitExceededCondition("c1")
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_DeviceID(b *testing.B) {
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = fmt.Sprintf("device_%03d", i)
	}
	cond := condition.NewDeviceIDCondition("c1", ids)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

func BenchmarkCondition_FQNPrefix(b *testing.B) {
	prefixes := []string{"device_001/volt", "device_002/curr", "device_003/pow"}
	cond := condition.NewFQNCondition("c1", prefixes)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cond.Evaluate(ctx, data)
	}
}

// ─────────────────────────────────────────────
//  8 种内置动作 Benchmark — 纯执行（零分配目标 < 200ns/op）
// ─────────────────────────────────────────────

func BenchmarkAction_Transform(b *testing.B) {
	scale := 0.1
	act := action.NewTransformAction("a1", &scale, nil, "kV")
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.value = 220.5 // 重置
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_Rename(b *testing.B) {
	act := action.NewRenameAction("a1", "voltage_kv")
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_Tag(b *testing.B) {
	act := action.NewTagAction("a1", map[string]string{"source": "scada", "level": "primary"})
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_Drop(b *testing.B) {
	act := action.NewDropAction("a1")
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_Route(b *testing.B) {
	act := action.NewRouteAction("a1", []string{"kafka", "influxdb"})
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.targets = data.targets[:0] // 重置
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_LimitCheck(b *testing.B) {
	act := action.NewLimitCheckAction("a1")
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.limitExceeded = false // 重置
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_QualityMark(b *testing.B) {
	act := action.NewQualityMarkAction("a1", 192, 0)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.quality = 192 // 重置
		_ = act.Execute(ctx, data)
	}
}

func BenchmarkAction_AlarmNotify(b *testing.B) {
	act := action.NewAlarmNotifyAction("a1", "warning", "voltage exceeded", func(_, _, _, _ string) {})
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = act.Execute(ctx, data)
	}
}

// ─────────────────────────────────────────────
//  编译器 Benchmark
// ─────────────────────────────────────────────

func BenchmarkCompile_SingleRule(b *testing.B) {
	rule := buildRule("r1", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiler.CompileRule(rule)
	}
}

func BenchmarkCompile_10RuleChain(b *testing.B) {
	chain := buildChain(10)
	reg := &noopRegistry{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiler.CompileChain(chain, reg)
	}
}

// ─────────────────────────────────────────────
//  引擎热路径 Benchmark
//  目标：单规则 < 100ns, 10 规则链 < 1μs
// ─────────────────────────────────────────────

func BenchmarkEngine_SingleRule(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(1)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.reset() // 重置数据点
		_, _ = engine.EvalChain(ctx, "bench", data)
	}
}

func BenchmarkEngine_5Rules(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(5)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.reset()
		_, _ = engine.EvalChain(ctx, "bench", data)
	}
}

func BenchmarkEngine_10Rules(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(10)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.reset()
		_, _ = engine.EvalChain(ctx, "bench", data)
	}
}

func BenchmarkEngine_50Rules(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(50)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.reset()
		_, _ = engine.EvalChain(ctx, "bench", data)
	}
}

func BenchmarkEngine_100Rules(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(100)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.reset()
		_, _ = engine.EvalChain(ctx, "bench", data)
	}
}

// ─────────────────────────────────────────────
//  条件树组合 Benchmark — 纯评估
// ─────────────────────────────────────────────

func BenchmarkConditionTree_AND3(b *testing.B) {
	root := &core.ConditionNode{
		Operator: core.OpAnd,
		Children: []*core.ConditionNode{
			{Leaf: condition.NewDeviceTypeCondition("c1", []string{"analog"})},
			{Leaf: condition.NewPointNameCondition("c2", []string{"voltage"})},
		},
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = root.Evaluate(ctx, data)
	}
}

func BenchmarkConditionTree_OR5(b *testing.B) {
	children := make([]*core.ConditionNode, 5)
	for i := 0; i < 5; i++ {
		children[i] = &core.ConditionNode{
			Leaf: condition.NewDeviceTypeCondition(fmt.Sprintf("c_%d", i), []string{"analog"}),
		}
	}
	root := &core.ConditionNode{Operator: core.OpOr, Children: children}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = root.Evaluate(ctx, data)
	}
}

func BenchmarkConditionTree_NOT(b *testing.B) {
	root := &core.ConditionNode{
		Operator: core.OpNot,
		Children: []*core.ConditionNode{
			{Leaf: condition.NewDeviceTypeCondition("c1", []string{"analog"})},
		},
	}
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = root.Evaluate(ctx, data)
	}
}

// ─────────────────────────────────────────────
//  编译后条件树/动作链 Benchmark（零分配验证）
// ─────────────────────────────────────────────

func BenchmarkCompiledConditionTree_AND3(b *testing.B) {
	root := &core.ConditionNode{
		Operator: core.OpAnd,
		Children: []*core.ConditionNode{
			{Leaf: condition.NewDeviceTypeCondition("c1", []string{"analog"})},
			{Leaf: condition.NewPointNameCondition("c2", []string{"voltage"})},
		},
	}
	evalFunc, _ := compiler.CompileConditionTree(root)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = evalFunc(ctx, data)
	}
}

func BenchmarkCompiledActionChain_3Actions(b *testing.B) {
	scale := 0.1
	chain := &core.ActionChain{
		Actions: []core.Action{
			action.NewTransformAction("t1", &scale, nil, "kV"),
			action.NewLimitCheckAction("lc1"),
			action.NewRouteAction("r1", []string{"default"}),
		},
	}
	execFunc, _, _ := compiler.CompileActionChain(chain)
	ctx := context.Background()
	data := newBenchData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.reset()
		_ = execFunc(ctx, data)
	}
}

// ─────────────────────────────────────────────
//  批量评估 Benchmark
// ─────────────────────────────────────────────

func BenchmarkEngine_Batch10(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(5)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataList := make([]core.DataContext, 10)
		for j := range dataList {
			dataList[j] = newBenchData()
		}
		_, _ = engine.EvalChainBatch(ctx, "bench", dataList)
	}
}

func BenchmarkEngine_Batch100(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(5)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataList := make([]core.DataContext, 100)
		for j := range dataList {
			dataList[j] = newBenchData()
		}
		_, _ = engine.EvalChainBatch(ctx, "bench", dataList)
	}
}

// ─────────────────────────────────────────────
//  COW 热加载 Benchmark
// ─────────────────────────────────────────────

func BenchmarkEngine_LoadChain(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.LoadChain(chain)
	}
}

func BenchmarkEngine_UnloadChain(b *testing.B) {
	b.StopTimer()
	engine := engine.NewEngine()
	chain := buildChain(10)
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_ = engine.LoadChain(chain)
		b.StartTimer()
		engine.UnloadChain("bench")
	}
}

// ─────────────────────────────────────────────
//  并发安全 Benchmark
// ─────────────────────────────────────────────

func BenchmarkEngine_ConcurrentEval(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(5)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		data := newBenchData()
		for pb.Next() {
			data.reset()
			_, _ = engine.EvalChain(ctx, "bench", data)
		}
	})
}

// ─────────────────────────────────────────────
//  不匹配路径 Benchmark（条件快速短路）
// ─────────────────────────────────────────────

func BenchmarkEngine_NoMatch(b *testing.B) {
	engine := engine.NewEngine()
	chain := buildChain(10)
	if err := engine.LoadChain(chain); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := &benchDataPoint{
		pointType: "nonexistent",
		tags:      make(map[string]string),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.EvalChain(ctx, "bench", data)
	}
}

// ─────────────────────────────────────────────
//  辅助
// ─────────────────────────────────────────────

func buildRule(id string, priority int) *core.Rule {
	scale := 0.1
	return &core.Rule{
		ID:       id,
		Name:     "rule " + id,
		Priority: priority,
		Enabled:  true,
		Condition: &core.ConditionNode{
			Leaf: condition.NewDeviceTypeCondition(id+"_c", []string{"analog"}),
		},
		Actions: &core.ActionChain{
			Actions: []core.Action{
				action.NewTransformAction(id+"_t", &scale, nil, "kV"),
				action.NewLimitCheckAction(id + "_lc"),
			},
		},
		Targets: []string{"default"},
	}
}

func buildChain(ruleCount int) *core.RuleChain {
	rules := make([]*core.Rule, ruleCount)
	for i := 0; i < ruleCount; i++ {
		rules[i] = buildRule(fmt.Sprintf("rule_%d", i), i+1)
	}
	return &core.RuleChain{
		ID:      "bench",
		Name:    "Benchmark Chain",
		Root:    true,
		Version: 1,
		Status:  "deployed",
		Rules:   rules,
	}
}

// noopRegistry 用于 benchmark（不通过 registry 创建条件/动作）
type noopRegistry struct{}

func (r *noopRegistry) CreateCondition(typeName, id string, config map[string]any) (core.Condition, error) {
	return nil, fmt.Errorf("noop")
}
func (r *noopRegistry) CreateAction(typeName, id string, config map[string]any) (core.Action, error) {
	return nil, fmt.Errorf("noop")
}

var _ = strings.TrimSpace
