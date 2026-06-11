package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/memorysink"
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

// ─────────────────────────────────────────────
//  NewEngine 测试
// ─────────────────────────────────────────────

func TestNewEngine_Default(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("engine should not be nil")
	}
	ids := e.GetChainIDs()
	if len(ids) != 0 {
		t.Errorf("new engine should have no chains, got %v", ids)
	}
}

func TestNewEngine_WithOptions(t *testing.T) {
	e := NewEngine(
		WithErrorStrategy(core.ErrorStrategyAbort),
		WithEvaluationMode(core.EvalModeFirst),
		WithActionTimeout(5*time.Second),
	)
	if e.errorStrategy != core.ErrorStrategyAbort {
		t.Error("error strategy not set")
	}
	if e.evalMode != core.EvalModeFirst {
		t.Error("eval mode not set")
	}
	if e.actionTimeout != 5*time.Second {
		t.Error("action timeout not set")
	}
}

// ─────────────────────────────────────────────
//  LoadChain / UnloadChain 测试
// ─────────────────────────────────────────────

func TestEngine_LoadChain(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID:      "test-chain",
		Version: 1,
		Rules: []*core.Rule{
			{
				ID: "r1", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
				Targets: []string{"default"},
			},
		},
	}
	if err := e.LoadChain(chain); err != nil {
		t.Fatalf("load chain error: %v", err)
	}
	ids := e.GetChainIDs()
	if len(ids) != 1 || ids[0] != "test-chain" {
		t.Errorf("expected [test-chain], got %v", ids)
	}
}

func TestEngine_UnloadChain(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "chain-1", Version: 1,
		Rules: []*core.Rule{
			{ID: "r1", Enabled: true, Priority: 1},
		},
	}
	e.LoadChain(chain)
	e.UnloadChain("chain-1")
	ids := e.GetChainIDs()
	if len(ids) != 0 {
		t.Errorf("after unload, should have no chains, got %v", ids)
	}
}

func TestEngine_LoadChain_COW(t *testing.T) {
	e := NewEngine()

	// 加载第一个链
	chain1 := &core.RuleChain{ID: "chain-1", Version: 1, Rules: []*core.Rule{
		{ID: "r1", Enabled: true, Priority: 1},
	}}
	e.LoadChain(chain1)

	// 加载第二个链，chain-1 应该还在
	chain2 := &core.RuleChain{ID: "chain-2", Version: 1, Rules: []*core.Rule{
		{ID: "r2", Enabled: true, Priority: 1},
	}}
	e.LoadChain(chain2)

	ids := e.GetChainIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 chains, got %d: %v", len(ids), ids)
	}
}

func TestEngine_LoadChain_UpdateExisting(t *testing.T) {
	e := NewEngine()

	chain := &core.RuleChain{ID: "chain-1", Version: 1, Rules: []*core.Rule{
		{ID: "r1", Enabled: true, Priority: 1},
	}}
	e.LoadChain(chain)

	// 更新同一链
	chainV2 := &core.RuleChain{ID: "chain-1", Version: 2, Rules: []*core.Rule{
		{ID: "r1", Enabled: true, Priority: 1},
		{ID: "r2", Enabled: true, Priority: 2},
	}}
	e.LoadChain(chainV2)

	ids := e.GetChainIDs()
	if len(ids) != 1 {
		t.Errorf("expected 1 chain after update, got %d", len(ids))
	}
}

// ─────────────────────────────────────────────
//  EvalChain 测试
// ─────────────────────────────────────────────

func TestEngine_EvalChain_MatchRule(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "match-all", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
				Targets: []string{"default"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, err := e.EvalChain(context.Background(), "test-chain", data)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(result.MatchedRules) != 1 {
		t.Errorf("expected 1 matched rule, got %d", len(result.MatchedRules))
	}
	if result.Dropped {
		t.Error("should not be dropped")
	}
	if data.TargetCount() != 1 || data.TargetAt(0) != "default" {
		t.Errorf("expected target 'default', got targets: %v", data.targets)
	}
}

func TestEngine_EvalChain_NoMatch(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "never-match", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return false }),
				},
				Actions: &core.ActionChain{},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, err := e.EvalChain(context.Background(), "test-chain", data)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(result.MatchedRules) != 0 {
		t.Errorf("expected 0 matched rules, got %d", len(result.MatchedRules))
	}
}

func TestEngine_EvalChain_ChainNotFound(t *testing.T) {
	e := NewEngine()
	data := newMockData()
	result, err := e.EvalChain(context.Background(), "nonexistent", data)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if result.Dropped {
		t.Error("should not be dropped")
	}
}

func TestEngine_EvalChain_MultipleRules(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "rule-1", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						core.ActionFunc(func(_ context.Context, data core.DataContext) error {
							data.SetValue(data.Value() + 10)
							return nil
						}),
					},
				},
				Targets: []string{"target-1"},
			},
			{
				ID: "rule-2", Enabled: true, Priority: 2,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						core.ActionFunc(func(_ context.Context, data core.DataContext) error {
							data.SetValue(data.Value() * 2)
							return nil
						}),
					},
				},
				Targets: []string{"target-2"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	data.value = 100
	result, err := e.EvalChain(context.Background(), "test-chain", data)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(result.MatchedRules) != 2 {
		t.Errorf("expected 2 matched rules, got %d", len(result.MatchedRules))
	}
	// 100 + 10 = 110, then 110 * 2 = 220
	if data.Value() != 220 {
		t.Errorf("expected value 220, got %f", data.Value())
	}
}

func TestEngine_EvalChain_EvalModeFirst(t *testing.T) {
	e := NewEngine(WithEvaluationMode(core.EvalModeFirst))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "rule-1", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
				Targets: []string{"first"},
			},
			{
				ID: "rule-2", Enabled: true, Priority: 2,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
				Targets: []string{"second"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, _ := e.EvalChain(context.Background(), "test-chain", data)
	if len(result.MatchedRules) != 1 {
		t.Errorf("EvalModeFirst should match only 1 rule, got %d", len(result.MatchedRules))
	}
}

func TestEngine_EvalChain_DropAction(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "drop-rule", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						core.ActionFunc(func(_ context.Context, _ core.DataContext) error { return core.ErrDropData }),
					},
				},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, err := e.EvalChain(context.Background(), "test-chain", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Dropped {
		t.Error("result should be dropped")
	}
	if !data.Dropped() {
		t.Error("data should be marked dropped")
	}
}

func TestEngine_EvalChain_ActionError_Continue(t *testing.T) {
	e := NewEngine(WithErrorStrategy(core.ErrorStrategyContinue))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "err-rule", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						core.ActionFunc(func(_ context.Context, _ core.DataContext) error { return errors.New("boom") }),
					},
				},
			},
			{
				ID: "ok-rule", Enabled: true, Priority: 2,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
				Targets: []string{"next"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, err := e.EvalChain(context.Background(), "test-chain", data)
	if err != nil {
		t.Fatalf("continue strategy should not return error: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if len(result.MatchedRules) != 2 {
		t.Errorf("expected 2 matched rules (continue after error), got %d", len(result.MatchedRules))
	}
}

func TestEngine_EvalChain_ActionError_Abort(t *testing.T) {
	e := NewEngine(WithErrorStrategy(core.ErrorStrategyAbort))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "err-rule", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						core.ActionFunc(func(_ context.Context, _ core.DataContext) error { return errors.New("boom") }),
					},
				},
			},
			{
				ID: "ok-rule", Enabled: true, Priority: 2,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
				Targets: []string{"next"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, err := e.EvalChain(context.Background(), "test-chain", data)
	if err == nil {
		t.Error("abort strategy should return error")
	}
	if len(result.MatchedRules) != 1 {
		t.Errorf("expected 1 matched rule (aborted), got %d", len(result.MatchedRules))
	}
}

// ─────────────────────────────────────────────
//  背压测试
// ─────────────────────────────────────────────

type mockBackpressure struct {
	level contract.Level
}

func (m *mockBackpressure) ShouldAccept(deviceID string) bool { return m.level < contract.Dropping }
func (m *mockBackpressure) CurrentLevel() contract.Level      { return m.level }

func TestEngine_EvalChain_BackpressureDropping(t *testing.T) {
	bp := &mockBackpressure{level: contract.Dropping}
	sink := memorysink.NewMemorySink()
	e := NewEngine(WithBackpressureIndicator(bp), WithMetricsSink(sink))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{ID: "r1", Enabled: true, Priority: 1},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, _ := e.EvalChain(context.Background(), "test-chain", data)
	if len(result.MatchedRules) != 0 {
		t.Error("dropping mode should skip all rules")
	}
	snap := sink.Snapshot()
	if snap.EvalTotal["test-chain"]["backpressure_dropped"] != 1 {
		t.Errorf("backpressure drop should be counted once, got %d", snap.EvalTotal["test-chain"]["backpressure_dropped"])
	}
}

func TestEngine_EvalChain_BackpressureDegraded(t *testing.T) {
	bp := &mockBackpressure{level: contract.Degraded}
	e := NewEngine(WithBackpressureIndicator(bp))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "high-pri", Enabled: true, Priority: 5,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"a"},
			},
			{
				ID: "low-pri", Enabled: true, Priority: 15,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"b"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, _ := e.EvalChain(context.Background(), "test-chain", data)
	if len(result.MatchedRules) != 1 {
		t.Errorf("degraded should skip low-pri, got %d matched", len(result.MatchedRules))
	}
	if result.MatchedRules[0].ID != "high-pri" {
		t.Errorf("expected high-pri rule, got %s", result.MatchedRules[0].ID)
	}
}

func TestEngine_EvalChain_BackpressurePaused(t *testing.T) {
	bp := &mockBackpressure{level: contract.Paused}
	e := NewEngine(WithBackpressureIndicator(bp))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "critical", Enabled: true, Priority: 0,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"a"},
			},
			{
				ID: "normal", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"b"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, _ := e.EvalChain(context.Background(), "test-chain", data)
	if len(result.MatchedRules) != 1 {
		t.Errorf("paused should only run priority 0, got %d matched", len(result.MatchedRules))
	}
	if result.MatchedRules[0].ID != "critical" {
		t.Errorf("expected critical rule, got %s", result.MatchedRules[0].ID)
	}
}

// ─────────────────────────────────────────────
//  DLQ 测试
// ─────────────────────────────────────────────

type mockDLQ struct {
	drops  []string
	errors []string
}

func (d *mockDLQ) TrackDrop(_ context.Context, _ any, ruleID string, reason string) {
	d.drops = append(d.drops, ruleID+":"+reason)
}
func (d *mockDLQ) TrackError(_ context.Context, _ any, ruleID string, err error) {
	d.errors = append(d.errors, ruleID+":"+err.Error())
}

func TestEngine_EvalChain_DLQ(t *testing.T) {
	dlq := &mockDLQ{}
	e := NewEngine(WithDataLossTracker(dlq))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "drop-rule", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions: &core.ActionChain{
					Actions: []core.Action{core.ActionFunc(func(_ context.Context, _ core.DataContext) error { return core.ErrDropData })},
				},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	e.EvalChain(context.Background(), "test-chain", data)
	if len(dlq.drops) == 0 {
		t.Error("DLQ should track drop")
	}
}

func TestEngine_EvalChain_DLQError(t *testing.T) {
	dlq := &mockDLQ{}
	e := NewEngine(WithDataLossTracker(dlq), WithErrorStrategy(core.ErrorStrategyContinue))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "err-rule", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions: &core.ActionChain{
					Actions: []core.Action{core.ActionFunc(func(_ context.Context, _ core.DataContext) error { return errors.New("fail") })},
				},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	e.EvalChain(context.Background(), "test-chain", data)
	if len(dlq.errors) == 0 {
		t.Error("DLQ should track error")
	}
}

// ─────────────────────────────────────────────
//  Panic Recovery 测试
// ─────────────────────────────────────────────

func TestEngine_EvalChain_PanicRecovery(t *testing.T) {
	e := NewEngine(WithErrorStrategy(core.ErrorStrategyContinue))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "panic-rule", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions: &core.ActionChain{
					Actions: []core.Action{core.ActionFunc(func(_ context.Context, _ core.DataContext) error {
						panic("oops!")
					})},
				},
			},
			{
				ID: "next-rule", Enabled: true, Priority: 2,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"ok"},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	result, _ := e.EvalChain(context.Background(), "test-chain", data)
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error from panic, got %d", len(result.Errors))
	}
	if len(result.MatchedRules) != 2 {
		t.Errorf("continue after panic, expected 2 matched, got %d", len(result.MatchedRules))
	}
}

// ─────────────────────────────────────────────
//  Metrics 测试
// ─────────────────────────────────────────────

func TestEngine_Metrics(t *testing.T) {
	sink := memorysink.NewMemorySink()
	e := NewEngine(WithMetricsSink(sink))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "r1", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{},
			},
		},
	}
	e.LoadChain(chain)

	data := newMockData()
	e.EvalChain(context.Background(), "test-chain", data)

	snap := sink.Snapshot()
	if snap.EvalCount["test-chain"] != 1 {
		t.Errorf("expected 1 evaluation, got %d", snap.EvalCount["test-chain"])
	}
	if snap.EvalTotal["test-chain"]["matched"] != 1 {
		t.Errorf("expected 1 match, got %d", snap.EvalTotal["test-chain"]["matched"])
	}
}

// ─────────────────────────────────────────────
//  EvalChainBatch 测试
// ─────────────────────────────────────────────

func TestEngine_EvalChainBatch(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "r1", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"default"},
			},
		},
	}
	e.LoadChain(chain)

	dataList := []core.DataContext{newMockData(), newMockData(), newMockData()}
	results, err := e.EvalChainBatch(context.Background(), "test-chain", dataList)
	if err != nil {
		t.Fatalf("batch eval error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if len(r.MatchedRules) != 1 {
			t.Errorf("result[%d]: expected 1 match, got %d", i, len(r.MatchedRules))
		}
	}
}

func TestEngine_EvalChainBatch_ChainNotFound(t *testing.T) {
	e := NewEngine()
	dataList := []core.DataContext{newMockData()}
	results, err := e.EvalChainBatch(context.Background(), "nonexistent", dataList)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ─────────────────────────────────────────────
//  并发安全测试
// ─────────────────────────────────────────────

func TestEngine_ConcurrentEval(t *testing.T) {
	sink := memorysink.NewMemorySink()
	e := NewEngine(WithMetricsSink(sink))
	chain := &core.RuleChain{
		ID: "test-chain", Version: 1,
		Rules: []*core.Rule{
			{
				ID: "r1", Enabled: true, Priority: 1,
				Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
				Actions:   &core.ActionChain{}, Targets: []string{"default"},
			},
		},
	}
	e.LoadChain(chain)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := newMockData()
			result, _ := e.EvalChain(context.Background(), "test-chain", data)
			if len(result.MatchedRules) != 1 {
				t.Errorf("concurrent eval: expected 1 match, got %d", len(result.MatchedRules))
			}
		}()
	}
	wg.Wait()

	snap := sink.Snapshot()
	if snap.EvalCount["test-chain"] != 100 {
		t.Errorf("expected 100 evaluations, got %d", snap.EvalCount["test-chain"])
	}
}

func TestEngine_ConcurrentLoadAndEval(t *testing.T) {
	e := NewEngine()
	var wg sync.WaitGroup

	// 并发加载和评估
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			chain := &core.RuleChain{
				ID: "test-chain", Version: idx,
				Rules: []*core.Rule{
					{ID: "r1", Enabled: true, Priority: 1},
				},
			}
			e.LoadChain(chain)
		}(i)
		go func() {
			defer wg.Done()
			data := newMockData()
			e.EvalChain(context.Background(), "test-chain", data)
		}()
	}
	wg.Wait()
}
