package tokenbucket

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/contrib/memorysink"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/engine"
)

// ─────────────────────────────────────────────
//  GlobalLimiter
// ─────────────────────────────────────────────

func TestGlobalLimiter_BurstThenDeny(t *testing.T) {
	// rate=1/s, burst=3：首 3 个立即放行，之后拒绝直到令牌补充
	lim := NewGlobalLimiter(1, 3)

	for i := 0; i < 3; i++ {
		if !lim.Allow("k1") {
			t.Errorf("burst %d should allow", i)
		}
	}
	if lim.Allow("k1") {
		t.Error("4th call should be denied (burst exhausted)")
	}
}

func TestGlobalLimiter_Refill(t *testing.T) {
	lim := NewGlobalLimiter(100, 1) // 100/s, burst=1
	if !lim.Allow("k") {
		t.Fatal("first should allow")
	}
	if lim.Allow("k") {
		t.Fatal("second should be denied immediately")
	}
	// 等待 20ms，足够补充 2 个令牌
	time.Sleep(20 * time.Millisecond)
	if !lim.Allow("k") {
		t.Error("after refill should allow")
	}
}

func TestGlobalLimiter_AllowIgnoresKey(t *testing.T) {
	// GlobalLimiter 所有 key 共享一个桶
	lim := NewGlobalLimiter(1, 2)
	lim.Allow("a")
	lim.Allow("b")
	if lim.Allow("c") {
		t.Error("global bucket should be exhausted regardless of key")
	}
}

func TestGlobalLimiter_Wait(t *testing.T) {
	lim := NewGlobalLimiter(100, 1) // 100/s
	lim.Allow("k")                  // 消耗令牌
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := lim.Wait(ctx, "k"); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	d := time.Since(start)
	if d < 5*time.Millisecond {
		t.Errorf("Wait should have blocked briefly, took %v", d)
	}
}

func TestGlobalLimiter_WaitCancelled(t *testing.T) {
	lim := NewGlobalLimiter(1, 1) // 1/s
	lim.Allow("k")                // 消耗令牌
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := lim.Wait(ctx, "k")
	if err == nil {
		t.Error("expected ctx error")
	}
}

func TestGlobalLimiter_ImplementsCoreInterface(t *testing.T) {
	var _ contract.Limiter = NewGlobalLimiter(10, 10)
}

// ─────────────────────────────────────────────
//  PerKeyLimiter
// ─────────────────────────────────────────────

func TestPerKeyLimiter_KeyIndependence(t *testing.T) {
	lim := NewPerKeyLimiter(1, 1)
	lim.Allow("a") // 消耗 a 的桶
	if !lim.Allow("b") {
		t.Error("per-key: b should be independent of a")
	}
}

func TestPerKeyLimiter_BurstExhaustion(t *testing.T) {
	lim := NewPerKeyLimiter(1, 2)
	lim.Allow("a")
	lim.Allow("a")
	if lim.Allow("a") {
		t.Error("burst 2 should be exhausted on 3rd call")
	}
}

func TestPerKeyLimiter_Concurrent(t *testing.T) {
	lim := NewPerKeyLimiter(1000, 1000)
	var wg sync.WaitGroup
	var allowed atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if lim.Allow("hot-key") {
					allowed.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	// burst=1000，所有 1000 个调用都应当放行
	if got := allowed.Load(); got != 1000 {
		t.Errorf("allowed: want 1000, got %d", got)
	}
}

func TestPerKeyLimiter_GC(t *testing.T) {
	lim := NewPerKeyLimiter(1, 1)
	lim.WithIdleTTL(50 * time.Millisecond).WithGCPeriod(30 * time.Millisecond)
	lim.Allow("a")
	lim.Allow("b")
	if lim.Size() != 2 {
		t.Errorf("want 2 keys, got %d", lim.Size())
	}
	// 等待 GC 触发
	time.Sleep(150 * time.Millisecond)
	lim.Allow("c") // 触发 GC
	if lim.Size() > 1 {
		t.Errorf("after GC, want <=1 keys, got %d", lim.Size())
	}
}

func TestPerKeyLimiter_ImplementsCoreInterface(t *testing.T) {
	var _ contract.Limiter = NewPerKeyLimiter(10, 10)
}

// ─────────────────────────────────────────────
//  与 Engine 集成
// ─────────────────────────────────────────────

// memDC 测试用 DataContext
type memDC struct{}

func (m *memDC) DeviceID() string                       { return "d1" }
func (m *memDC) PointName() string                      { return "p1" }
func (m *memDC) PointType() string                      { return "analog" }
func (m *memDC) FQN() string                            { return "d1/p1" }
func (m *memDC) Value() float64                         { return 1.0 }
func (m *memDC) SetValue(_ float64)                     {}
func (m *memDC) Quality() int                           { return 0 }
func (m *memDC) SetQuality(_ int)                       {}
func (m *memDC) UpperLimit() (float64, bool)            { return 0, false }
func (m *memDC) LowerLimit() (float64, bool)            { return 0, false }
func (m *memDC) LimitExceeded() bool                    { return false }
func (m *memDC) SetLimitExceeded(_ bool)                {}
func (m *memDC) GetTag(_ string) string                 { return "" }
func (m *memDC) SetTag(_, _ string)                     {}
func (m *memDC) TargetCount() int                       { return 0 }
func (m *memDC) TargetAt(_ int) string                  { return "" }
func (m *memDC) AddTarget(_ string)                     {}
func (m *memDC) Dropped() bool                          { return false }
func (m *memDC) SetDropped(_ bool)                      {}
func (m *memDC) Timestamp() int64                       { return time.Now().UnixNano() }
func (m *memDC) SpanContext() contract.SpanContext     { return contract.SpanContext{} }
func (m *memDC) SetSpanContext(_ contract.SpanContext) {}
func (m *memDC) PreviousValue() (float64, bool)         { return 0, false }
func (m *memDC) SetPreviousValue(_ float64)             {}
func (m *memDC) Raw() any                               { return nil }

func TestEngine_Integration_WithPerKeyLimiter(t *testing.T) {
	lim := NewPerKeyLimiter(1, 1) // 每 key 每秒 1 个
	sink := memorysink.NewMemorySink()

	e := engine.NewEngine(
		engine.WithLimiter(lim),
		engine.WithMetricsSink(sink),
	)

	e.LoadChain(&core.RuleChain{
		ID: "c1",
		Rules: []*core.Rule{{
			ID: "r1", Enabled: true, Priority: 1,
			Condition: &core.ConditionNode{Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true })},
			Actions:   &core.ActionChain{},
		}},
	})

	// 第一次：放行
	r1, _ := e.EvalChain(context.Background(), "c1", &memDC{})
	if len(r1.MatchedRules) != 1 {
		t.Errorf("1st: want 1 match, got %d", len(r1.MatchedRules))
	}

	// 第二次：限流
	r2, _ := e.EvalChain(context.Background(), "c1", &memDC{})
	if len(r2.MatchedRules) != 0 {
		t.Errorf("2nd: want 0 match (throttled), got %d", len(r2.MatchedRules))
	}
	if sink.RuleThrottled["c1"]["r1"] != 1 {
		t.Errorf("RuleThrottled: want 1, got %d", sink.RuleThrottled["c1"]["r1"])
	}

	// 等令牌补充
	time.Sleep(1100 * time.Millisecond)
	r3, _ := e.EvalChain(context.Background(), "c1", &memDC{})
	if len(r3.MatchedRules) != 1 {
		t.Errorf("3rd (after refill): want 1 match, got %d", len(r3.MatchedRules))
	}
}
