package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  测试辅助：内存数据上下文
// ─────────────────────────────────────────────

// memDataContext 测试用内存数据上下文
type memDataContext struct {
	deviceID  string
	pointName string
	value     float64
	quality   int
	dropped   bool
	tags      map[string]string
	targets   []string
}

func newMemDC(deviceID, pointName string, value float64) *memDataContext {
	return &memDataContext{
		deviceID:  deviceID,
		pointName: pointName,
		value:     value,
		tags:      make(map[string]string),
	}
}

func (m *memDataContext) DeviceID() string                      { return m.deviceID }
func (m *memDataContext) PointName() string                     { return m.pointName }
func (m *memDataContext) SetPointName(name string)              { m.pointName = name }
func (m *memDataContext) PointType() string                     { return "analog" }
func (m *memDataContext) FQN() string                           { return m.deviceID + "/" + m.pointName }
func (m *memDataContext) Value() float64                        { return m.value }
func (m *memDataContext) SetValue(v float64)                    { m.value = v }
func (m *memDataContext) Quality() int                          { return m.quality }
func (m *memDataContext) SetQuality(q int)                      { m.quality = q }
func (m *memDataContext) UpperLimit() (float64, bool)           { return 0, false }
func (m *memDataContext) LowerLimit() (float64, bool)           { return 0, false }
func (m *memDataContext) LimitExceeded() bool                   { return false }
func (m *memDataContext) SetLimitExceeded(v bool)               {}
func (m *memDataContext) GetTag(key string) string              { return m.tags[key] }
func (m *memDataContext) SetTag(key, value string)              { m.tags[key] = value }
func (m *memDataContext) TargetCount() int                      { return len(m.targets) }
func (m *memDataContext) TargetAt(i int) string                 { return m.targets[i] }
func (m *memDataContext) AddTarget(target string)               { m.targets = append(m.targets, target) }
func (m *memDataContext) Dropped() bool                         { return m.dropped }
func (m *memDataContext) SetDropped(v bool)                     { m.dropped = v }
func (m *memDataContext) Timestamp() int64                      { return time.Now().UnixNano() }
func (m *memDataContext) SpanContext() contract.SpanContext     { return contract.SpanContext{} }
func (m *memDataContext) SetSpanContext(_ contract.SpanContext) {}
func (m *memDataContext) PreviousValue() (float64, bool)        { return 0, false }
func (m *memDataContext) SetPreviousValue(_ float64)            {}
func (m *memDataContext) Raw() any                              { return nil }

// ─────────────────────────────────────────────
//  Shutdown 测试
// ─────────────────────────────────────────────

func TestShutdown_MarkShutdown_Once(t *testing.T) {
	s := newShutdown()
	if !s.markShutdown() {
		t.Error("first markShutdown should return true")
	}
	if s.markShutdown() {
		t.Error("second markShutdown should return false")
	}
}

func TestShutdown_Begin_AfterShutdown(t *testing.T) {
	s := newShutdown()
	s.markShutdown()
	if s.begin() {
		t.Error("begin should return false after shutdown")
	}
}

func TestShutdown_Begin_BeforeShutdown(t *testing.T) {
	s := newShutdown()
	if !s.begin() {
		t.Error("begin should return true before shutdown")
	}
	s.end()
}

func TestShutdown_Wait_Completes(t *testing.T) {
	s := newShutdown()

	// 模拟在 shutdown 之前已经开始的评估
	if !s.begin() {
		t.Fatal("begin should succeed before shutdown")
	}

	// 标记关闭
	s.markShutdown()

	// 异步结束评估
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.end()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := s.wait(ctx); err != nil {
		t.Errorf("wait should complete normally, got: %v", err)
	}
	if !s.isShutdown() {
		t.Error("state should be ShutdownStateShutdown")
	}
}

func TestShutdown_Wait_Timeout(t *testing.T) {
	s := newShutdown()

	// 模拟永远不完成的评估（在 shutdown 之前开始）
	if !s.begin() {
		t.Fatal("begin should succeed before shutdown")
	}
	s.markShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.wait(ctx)
	if err == nil {
		t.Error("wait should timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestEngine_Shutdown_RejectsNewEval(t *testing.T) {
	e := NewEngine()
	e.LoadChain(&core.RuleChain{ID: "c1", Rules: []*core.Rule{}})

	if err := e.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	_, err := e.EvalChain(context.Background(), "c1", newMemDC("d1", "p1", 1.0))
	if err == nil {
		t.Error("expected error after shutdown")
	}
	if !errors.Is(err, core.ErrEngineShutdown) {
		t.Errorf("expected ErrEngineShutdown, got: %v", err)
	}
}

// ─────────────────────────────────────────────
//  健康检查测试
// ─────────────────────────────────────────────

func TestEngine_HealthCheck_Basic(t *testing.T) {
	e := NewEngine()
	// 等待至少 1 毫秒以确保 uptime 非零
	time.Sleep(1 * time.Millisecond)
	hs := e.HealthCheck()

	if hs.Status != "healthy" {
		t.Errorf("expected healthy, got %s", hs.Status)
	}
	if hs.LoadedChains != 0 {
		t.Errorf("expected 0 chains, got %d", hs.LoadedChains)
	}
	if hs.UptimeSeconds < 0 {
		t.Errorf("uptime_seconds should not be negative, got %d", hs.UptimeSeconds)
	}
	if hs.ShuttingDown {
		t.Error("should not be shutting down")
	}
}

func TestEngine_HealthCheck_AfterLoadChains(t *testing.T) {
	e := NewEngine()
	e.LoadChain(&core.RuleChain{ID: "c1", Rules: []*core.Rule{}})
	e.LoadChain(&core.RuleChain{ID: "c2", Rules: []*core.Rule{}})

	hs := e.HealthCheck()
	if hs.LoadedChains != 2 {
		t.Errorf("expected 2 chains, got %d", hs.LoadedChains)
	}
}

func TestEngine_HealthCheck_DuringShutdown(t *testing.T) {
	e := NewEngine()
	e.Shutdown(context.Background())

	hs := e.HealthCheck()
	if !hs.ShuttingDown {
		t.Error("should be shutting down")
	}
	if hs.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", hs.Status)
	}
}

func TestEngine_HealthCheck_ImplementsInterface(t *testing.T) {
	// V4.4：HealthChecker 本地接口已删除（与 contract.StatusReporter 重复）；
	//       改用 contract.StatusReporter 校验实现。
	var _ contract.StatusReporter = (*Engine)(nil)
	var _ contract.LivenessChecker = (*Engine)(nil)
	var _ contract.ReadinessChecker = (*Engine)(nil)
	var _ contract.ChainLister = (*Engine)(nil)
	var _ contract.HealthProvider = (*Engine)(nil)
}

// ─────────────────────────────────────────────
//  引擎集成测试：依赖图 + LoadChain
// ─────────────────────────────────────────────

func TestEngine_LoadChain_CyclicDependency(t *testing.T) {
	e := NewEngine()

	// 先加载 chain A 引用 B
	chainA := &core.RuleChain{
		ID:    "A",
		Rules: []*core.Rule{},
		Refs:  []string{"B"},
	}
	if err := e.LoadChain(chainA); err != nil {
		t.Fatalf("load A failed: %v", err)
	}

	// 加载 chain B 引用 A → 循环
	chainB := &core.RuleChain{
		ID:    "B",
		Rules: []*core.Rule{},
		Refs:  []string{"A"},
	}
	err := e.LoadChain(chainB)
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
	if !errors.Is(err, core.ErrCyclicDependency) {
		t.Errorf("expected ErrCyclicDependency, got: %v", err)
	}

	// B 不应被加载
	ids := e.GetChainIDs()
	if len(ids) != 1 || ids[0] != "A" {
		t.Errorf("expected only [A], got %v", ids)
	}
}

func TestEngine_LoadChain_SelfReference(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID:    "self",
		Rules: []*core.Rule{},
		Refs:  []string{"self"},
	}
	err := e.LoadChain(chain)
	if err == nil {
		t.Fatal("expected error for self-reference")
	}
	if !errors.Is(err, core.ErrCyclicDependency) {
		t.Errorf("expected ErrCyclicDependency, got: %v", err)
	}
}

func TestEngine_UnloadChain_CleanupDepGraph(t *testing.T) {
	e := NewEngine()
	e.LoadChain(&core.RuleChain{ID: "A", Rules: []*core.Rule{}})
	e.LoadChain(&core.RuleChain{ID: "B", Refs: []string{"A"}, Rules: []*core.Rule{}})

	// V4.7：depGraph 已改为 compiler.DependencyGraph 实例
	if !e.depGraph.HasNode("A") || !e.depGraph.HasNode("B") {
		t.Fatal("both A and B should be in dep graph")
	}

	e.UnloadChain("A")

	if e.depGraph.HasNode("A") {
		t.Error("A should be removed from dep graph")
	}
	if !e.depGraph.HasNode("B") {
		t.Error("B should still be in dep graph")
	}
}

// ─────────────────────────────────────────────
//  引擎集成测试：Panic Recovery
// ─────────────────────────────────────────────

func TestEngine_EvalChain_ConditionPanicRecovery(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "panic-chain",
		Rules: []*core.Rule{
			{
				ID:       "panic-rule",
				Enabled:  true,
				Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool {
						panic("boom!")
					}),
				},
				Actions: &core.ActionChain{},
			},
		},
	}
	if err := e.LoadChain(chain); err != nil {
		t.Fatalf("load chain failed: %v", err)
	}

	// 顶层 panic 应当被 recover
	result, err := e.EvalChain(context.Background(), "panic-chain", newMemDC("d1", "p1", 1.0))
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	if result == nil {
		t.Error("result should not be nil")
	}
	if !errors.Is(err, core.ErrPanicRecovered) {
		t.Errorf("expected ErrPanicRecovered, got: %v", err)
	}
	var rfe *core.RuleFlowError
	if !errors.As(err, &rfe) {
		t.Error("error should be RuleFlowError")
	}
	if rfe != nil && rfe.Type != core.ErrorTypePanic {
		t.Errorf("expected ErrorTypePanic, got %v", rfe.Type)
	}
}

// ─────────────────────────────────────────────
//  引擎集成测试：动作 panic recovery
// ─────────────────────────────────────────────

func TestEngine_Action_PanicRecovery(t *testing.T) {
	e := NewEngine()
	chain := &core.RuleChain{
		ID: "panic-action",
		Rules: []*core.Rule{
			{
				ID:       "rule1",
				Enabled:  true,
				Priority: 1,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						core.ActionFunc(func(_ context.Context, _ core.DataContext) error {
							panic("action boom")
						}),
					},
				},
			},
		},
	}
	if err := e.LoadChain(chain); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	result, err := e.EvalChain(context.Background(), "panic-action", newMemDC("d1", "p1", 1.0))
	if err != nil {
		t.Errorf("EvalChain should not return error for action panic (handled internally), got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if len(result.Errors) == 0 {
		t.Error("expected error in result.Errors")
	}
	// 动作 panic 在 executeWithGuard 内部恢复，应被记录在 result.Errors
	if !strings.Contains(result.Errors[0].Message, "panic") {
		t.Errorf("error message should mention panic, got: %s", result.Errors[0].Message)
	}
}
