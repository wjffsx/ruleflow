package compiler

import (
	"context"
	"errors"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  mockDataContext — 测试用 DataContext 实现
// ─────────────────────────────────────────────

type mockDataContext struct {
	deviceID      string
	pointName     string
	pointType     string
	value         float64
	quality       int
	upperLimit    float64
	hasUpper      bool
	lowerLimit    float64
	hasLower      bool
	limitExceeded bool
	tags          map[string]string
	targets       []string
	dropped       bool
	timestamp     int64
}

func newMockData() *mockDataContext {
	return &mockDataContext{
		deviceID:  "device-001",
		pointName: "voltage",
		pointType: "analog",
		value:     220.5,
		quality:   192,
		tags:      make(map[string]string),
	}
}

func (m *mockDataContext) DeviceID() string                       { return m.deviceID }
func (m *mockDataContext) PointName() string                      { return m.pointName }
func (m *mockDataContext) PointType() string                      { return m.pointType }
func (m *mockDataContext) FQN() string                            { return m.deviceID + "/" + m.pointName }
func (m *mockDataContext) Value() float64                         { return m.value }
func (m *mockDataContext) SetValue(v float64)                     { m.value = v }
func (m *mockDataContext) Quality() int                           { return m.quality }
func (m *mockDataContext) SetQuality(q int)                       { m.quality = q }
func (m *mockDataContext) UpperLimit() (float64, bool)            { return m.upperLimit, m.hasUpper }
func (m *mockDataContext) LowerLimit() (float64, bool)            { return m.lowerLimit, m.hasLower }
func (m *mockDataContext) LimitExceeded() bool                    { return m.limitExceeded }
func (m *mockDataContext) SetLimitExceeded(v bool)                { m.limitExceeded = v }
func (m *mockDataContext) GetTag(key string) string               { return m.tags[key] }
func (m *mockDataContext) SetTag(key, value string)               { m.tags[key] = value }
func (m *mockDataContext) TargetCount() int                       { return len(m.targets) }
func (m *mockDataContext) TargetAt(i int) string                  { return m.targets[i] }
func (m *mockDataContext) AddTarget(target string)                { m.targets = append(m.targets, target) }
func (m *mockDataContext) Dropped() bool                          { return m.dropped }
func (m *mockDataContext) SetDropped(v bool)                      { m.dropped = v }
func (m *mockDataContext) Timestamp() int64                       { return m.timestamp }
func (m *mockDataContext) SpanContext() contract.SpanContext      { return contract.SpanContext{} }
func (m *mockDataContext) SetSpanContext(sc contract.SpanContext) {}
func (m *mockDataContext) Raw() any                               { return nil }
func (m *mockDataContext) PreviousValue() (float64, bool)         { return 0, false }
func (m *mockDataContext) SetPreviousValue(v float64)             {}

// mockAction 测试用 Action
type mockAction struct {
	typeVal string
}

func (a *mockAction) Execute(_ context.Context, _ core.DataContext) error { return nil }
func (a *mockAction) ID() string                                          { return "mock" }
func (a *mockAction) Type() string                                        { return a.typeVal }
func (a *mockAction) Description() string                                 { return "mock" }

// mockCondition 测试用 Condition
type mockCondition struct {
	typeVal string
}

func (c *mockCondition) Evaluate(_ context.Context, _ core.DataContext) bool { return true }
func (c *mockCondition) ID() string                                          { return "mock" }
func (c *mockCondition) Type() string                                        { return c.typeVal }
func (c *mockCondition) Description() string                                 { return "mock" }

// ─────────────────────────────────────────────
//  CompileConditionTree 测试
// ─────────────────────────────────────────────

func TestCompileConditionTree_Nil(t *testing.T) {
	f, err := CompileConditionTree(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f(context.Background(), newMockData()) {
		t.Error("nil condition tree should evaluate to true")
	}
}

func TestCompileConditionTree_Leaf(t *testing.T) {
	leaf := &core.ConditionNode{
		Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
	}
	f, err := CompileConditionTree(leaf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f(context.Background(), newMockData()) {
		t.Error("leaf condition should evaluate to true")
	}
}

func TestCompileConditionTree_And(t *testing.T) {
	node := &core.ConditionNode{
		Operator: core.OpAnd,
		Children: []*core.ConditionNode{
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return false })},
		},
	}
	f, err := CompileConditionTree(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f(context.Background(), newMockData()) {
		t.Error("AND with one false should evaluate to false")
	}
}

func TestCompileConditionTree_Or(t *testing.T) {
	node := &core.ConditionNode{
		Operator: core.OpOr,
		Children: []*core.ConditionNode{
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return false })},
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
		},
	}
	f, err := CompileConditionTree(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f(context.Background(), newMockData()) {
		t.Error("OR with one true should evaluate to true")
	}
}

func TestCompileConditionTree_Not(t *testing.T) {
	node := &core.ConditionNode{
		Operator: core.OpNot,
		Children: []*core.ConditionNode{
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
		},
	}
	f, err := CompileConditionTree(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f(context.Background(), newMockData()) {
		t.Error("NOT true should evaluate to false")
	}
}

func TestCompileConditionTree_UnknownOperator(t *testing.T) {
	node := &core.ConditionNode{
		Operator: core.LogicalOperator(99),
		Children: []*core.ConditionNode{
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
		},
	}
	_, err := CompileConditionTree(node)
	if err == nil {
		t.Error("unknown operator should return error")
	}
}

func TestCompileConditionTree_Nested(t *testing.T) {
	// (true AND (NOT false))
	root := &core.ConditionNode{
		Operator: core.OpAnd,
		Children: []*core.ConditionNode{
			{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
			{
				Operator: core.OpNot,
				Children: []*core.ConditionNode{
					{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return false })},
				},
			},
		},
	}
	f, err := CompileConditionTree(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f(context.Background(), newMockData()) {
		t.Error("(true AND (NOT false)) should evaluate to true")
	}
}

// ─────────────────────────────────────────────
//  CompileActionChain 测试
// ─────────────────────────────────────────────

func TestCompileActionChain_Nil(t *testing.T) {
	f, fast, err := CompileActionChain(nil)
	if err != nil || f != nil || !fast {
		t.Errorf("nil chain: fast=%v, err=%v", fast, err)
	}
}

func TestCompileActionChain_Empty(t *testing.T) {
	f, fast, err := CompileActionChain(&core.ActionChain{})
	if err != nil || f != nil || !fast {
		t.Errorf("empty chain: fast=%v, err=%v", fast, err)
	}
}

func TestCompileActionChain_Execute(t *testing.T) {
	chain := &core.ActionChain{
		Actions: []core.Action{
			core.ActionFunc(func(_ context.Context, data core.DataContext) error {
				data.SetValue(data.Value() + 1)
				return nil
			}),
		},
	}
	f, _, err := CompileActionChain(chain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := newMockData()
	data.value = 10
	if err := f(context.Background(), data); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if data.Value() != 11 {
		t.Errorf("expected 11, got %f", data.Value())
	}
}

func TestCompileActionChain_DropData(t *testing.T) {
	chain := &core.ActionChain{
		Actions: []core.Action{
			core.ActionFunc(func(_ context.Context, _ core.DataContext) error {
				return core.ErrDropData
			}),
		},
	}
	f, _, err := CompileActionChain(chain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := newMockData()
	if err := f(context.Background(), data); err != nil {
		t.Fatalf("drop should be handled: %v", err)
	}
	if !data.Dropped() {
		t.Error("data should be dropped")
	}
}

func TestCompileActionChain_Error(t *testing.T) {
	expectedErr := errors.New("boom")
	chain := &core.ActionChain{
		Actions: []core.Action{
			core.ActionFunc(func(_ context.Context, _ core.DataContext) error {
				return expectedErr
			}),
		},
	}
	f, _, err := CompileActionChain(chain)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	data := newMockData()
	if err := f(context.Background(), data); err != expectedErr {
		t.Errorf("expected boom, got %v", err)
	}
}

func TestCompileActionChain_FastPath(t *testing.T) {
	// 使用 FastPath 兼容的动作
	chain := &core.ActionChain{
		Actions: []core.Action{
			&mockAction{typeVal: "transform"},
			&mockAction{typeVal: "tag"},
		},
	}
	_, fast, err := CompileActionChain(chain)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !fast {
		t.Error("should be fast path")
	}

	// 使用非 FastPath 动作
	chain2 := &core.ActionChain{
		Actions: []core.Action{
			&mockAction{typeVal: "transform"},
			&mockAction{typeVal: "custom_slow"},
		},
	}
	_, fast2, err := CompileActionChain(chain2)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if fast2 {
		t.Error("should NOT be fast path")
	}
}

// ─────────────────────────────────────────────
//  CompileRule 测试
// ─────────────────────────────────────────────

func TestCompileRule_FastPath(t *testing.T) {
	rule := &core.Rule{
		ID:       "r1",
		Enabled:  true,
		Priority: 1,
		Condition: &core.ConditionNode{
			Leaf: &mockCondition{typeVal: "device_type"},
		},
		Actions: &core.ActionChain{
			Actions: []core.Action{&mockAction{typeVal: "transform"}},
		},
	}
	cr, err := CompileRule(rule)
	if err != nil {
		t.Fatalf("compile rule error: %v", err)
	}
	if !cr.IsFast {
		t.Error("should be fast path")
	}
	if cr.EvaluateFunc == nil || cr.ExecuteFunc == nil {
		t.Error("compiled functions should not be nil")
	}
}

func TestCompileRule_SlowPath(t *testing.T) {
	rule := &core.Rule{
		ID:       "r2",
		Enabled:  true,
		Priority: 1,
		Condition: &core.ConditionNode{
			Leaf: &mockCondition{typeVal: "custom_slow"},
		},
		Actions: &core.ActionChain{
			Actions: []core.Action{&mockAction{typeVal: "custom_slow"}},
		},
	}
	cr, err := CompileRule(rule)
	if err != nil {
		t.Fatalf("compile rule error: %v", err)
	}
	if cr.IsFast {
		t.Error("should NOT be fast path")
	}
}

func TestCompileRule_DepthExceeded(t *testing.T) {
	// 构建深度超限的条件树
	deep := &core.ConditionNode{Leaf: &mockCondition{typeVal: "device_type"}}
	for i := 0; i < MaxConditionDepth+1; i++ {
		deep = &core.ConditionNode{Operator: core.OpAnd, Children: []*core.ConditionNode{deep}}
	}
	rule := &core.Rule{
		ID:        "r-deep",
		Enabled:   true,
		Priority:  1,
		Condition: deep,
	}
	_, err := CompileRule(rule)
	if err == nil {
		t.Error("should fail for condition tree too deep")
	}
}

// ─────────────────────────────────────────────
//  CompileChain 测试
// ─────────────────────────────────────────────

func TestCompileChain_PrioritySort(t *testing.T) {
	chain := &core.RuleChain{
		ID:      "chain-1",
		Version: 1,
		Rules: []*core.Rule{
			{ID: "r3", Enabled: true, Priority: 30, Condition: &core.ConditionNode{Leaf: &mockCondition{typeVal: "device_type"}}},
			{ID: "r1", Enabled: true, Priority: 10, Condition: &core.ConditionNode{Leaf: &mockCondition{typeVal: "device_type"}}},
			{ID: "r2", Enabled: true, Priority: 20, Condition: &core.ConditionNode{Leaf: &mockCondition{typeVal: "device_type"}}},
		},
	}
	cc, err := CompileChain(chain, &core.DefaultRegistry{})
	if err != nil {
		t.Fatalf("compile chain error: %v", err)
	}
	if len(cc.SortedRules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(cc.SortedRules))
	}
	// 验证排序：Priority 升序
	if cc.SortedRules[0].Rule.ID != "r1" || cc.SortedRules[1].Rule.ID != "r2" || cc.SortedRules[2].Rule.ID != "r3" {
		t.Errorf("rules not sorted by priority: %s %s %s",
			cc.SortedRules[0].Rule.ID, cc.SortedRules[1].Rule.ID, cc.SortedRules[2].Rule.ID)
	}
}

func TestCompileChain_SkipDisabled(t *testing.T) {
	chain := &core.RuleChain{
		ID:      "chain-1",
		Version: 1,
		Rules: []*core.Rule{
			{ID: "r1", Enabled: true, Priority: 1, Condition: &core.ConditionNode{Leaf: &mockCondition{typeVal: "device_type"}}},
			{ID: "r2", Enabled: false, Priority: 2},
		},
	}
	cc, err := CompileChain(chain, &core.DefaultRegistry{})
	if err != nil {
		t.Fatalf("compile chain error: %v", err)
	}
	if len(cc.SortedRules) != 1 {
		t.Errorf("expected 1 rule (disabled skipped), got %d", len(cc.SortedRules))
	}
}

func TestCompileChain_FastSlowSplit(t *testing.T) {
	chain := &core.RuleChain{
		ID:      "chain-1",
		Version: 1,
		Rules: []*core.Rule{
			{ID: "fast", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: &mockCondition{typeVal: "device_type"}},
				Actions:   &core.ActionChain{Actions: []core.Action{&mockAction{typeVal: "transform"}}}},
			{ID: "slow", Enabled: true, Priority: 2,
				Condition: &core.ConditionNode{Leaf: &mockCondition{typeVal: "custom_slow"}},
				Actions:   &core.ActionChain{Actions: []core.Action{&mockAction{typeVal: "transform"}}}},
		},
	}
	cc, err := CompileChain(chain, &core.DefaultRegistry{})
	if err != nil {
		t.Fatalf("compile chain error: %v", err)
	}
	if len(cc.FastRules) != 1 || cc.FastRules[0].Rule.ID != "fast" {
		t.Errorf("expected 1 fast rule 'fast', got %v", cc.FastRules)
	}
	if len(cc.SlowRules) != 1 || cc.SlowRules[0].Rule.ID != "slow" {
		t.Errorf("expected 1 slow rule 'slow', got %v", cc.SlowRules)
	}
}
