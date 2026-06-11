package debug

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  mockSink — 测试用内存输出（线程安全）
// ─────────────────────────────────────────────

type mockSink struct {
	mu     sync.Mutex
	events []DebugEvent
	count  atomic.Int64
}

func (s *mockSink) WriteEvent(_ any, event DebugEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	s.count.Add(1)
	return nil
}

func (s *mockSink) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

// ─────────────────────────────────────────────
//  Tests
// ─────────────────────────────────────────────

func TestDebugModeString(t *testing.T) {
	tests := []struct {
		mode DebugMode
		want string
	}{
		{DebugOff, "off"},
		{DebugAll, "all"},
		{DebugFailures, "failures"},
		{DebugMode(99), "off"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("DebugMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestNewDebugManagerDefaults(t *testing.T) {
	dm := NewDebugManager(DebugOff, nil, time.Time{})
	if dm.Mode() != DebugOff {
		t.Errorf("expected DebugOff, got %v", dm.Mode())
	}
	if dm.Enabled() {
		t.Error("expected disabled when sink is nil")
	}
	// V4.9：限流已迁出至 contrib/debug/ratelimit_sink（不在 DebugManager 内部）
}

func TestModeSetMode(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugOff, sink, time.Time{})

	if dm.Mode() != DebugOff {
		t.Errorf("expected DebugOff, got %v", dm.Mode())
	}
	if dm.Enabled() {
		t.Error("expected disabled")
	}

	dm.SetMode(DebugAll, time.Time{})
	if dm.Mode() != DebugAll {
		t.Errorf("expected DebugAll, got %v", dm.Mode())
	}
	if !dm.Enabled() {
		t.Error("expected enabled")
	}

	dm.SetMode(DebugFailures, time.Time{})
	if dm.Mode() != DebugFailures {
		t.Errorf("expected DebugFailures, got %v", dm.Mode())
	}

	dm.SetMode(DebugOff, time.Time{})
	if dm.Mode() != DebugOff {
		t.Errorf("expected DebugOff, got %v", dm.Mode())
	}
	if dm.Enabled() {
		t.Error("expected disabled")
	}
}

func TestDeadlineAutoDisable(t *testing.T) {
	sink := &mockSink{}
	deadline := time.Now().Add(50 * time.Millisecond)
	dm := NewDebugManager(DebugAll, sink, deadline)

	if dm.Mode() != DebugAll {
		t.Errorf("expected DebugAll before deadline")
	}

	// 等待 deadline 过期
	time.Sleep(100 * time.Millisecond)

	// Mode() 是纯 getter，不会自动关闭；
	// deadline 过期检测由 Capture 触发。
	if dm.Mode() != DebugAll {
		t.Errorf("expected Mode() to still return DebugAll (pure getter), got %v", dm.Mode())
	}

	// 调用 Capture 触发 deadline 过期自动关闭
	dm.Capture(context.Background(), contract.DebugEvent{
		EventType:    contract.DebugEventIn,
		ChainID:      "test",
		RelationType: "matched",
	})

	if dm.Mode() != DebugOff {
		t.Errorf("expected DebugOff after Capture triggers deadline check, got %v", dm.Mode())
	}
	if dm.Enabled() {
		t.Error("expected disabled after deadline")
	}
}

func TestDebugFailuresOnlyCapturesErrorsAndDropped(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugFailures, sink, time.Time{})

	ctx := context.Background()
	dm.CaptureOut(ctx, "chain1", "rule1", "node1", "condition", "matched", `{"v":1}`, 100, "")
	dm.CaptureOut(ctx, "chain1", "rule1", "node1", "condition", "unmatched", `{"v":1}`, 100, "")
	dm.CaptureOut(ctx, "chain1", "rule1", "node1", "condition", "error", `{"v":1}`, 100, "something went wrong")
	dm.CaptureOut(ctx, "chain1", "rule1", "node1", "condition", "dropped", `{"v":1}`, 100, "")

	if sink.Len() != 2 {
		t.Errorf("expected 2 events (error+dropped), got %d", sink.Len())
	}
}

func TestShouldCapture(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugFailures, sink, time.Time{})

	if dm.ShouldCapture("matched") {
		t.Error("expected false for matched in DebugFailures mode")
	}
	if dm.ShouldCapture("unmatched") {
		t.Error("expected false for unmatched in DebugFailures mode")
	}
	if !dm.ShouldCapture("error") {
		t.Error("expected true for error in DebugFailures mode")
	}
	if !dm.ShouldCapture("dropped") {
		t.Error("expected true for dropped in DebugFailures mode")
	}

	// DebugAll 模式
	dm.SetMode(DebugAll, time.Time{})
	if !dm.ShouldCapture("matched") {
		t.Error("expected true for matched in DebugAll mode")
	}
	if !dm.ShouldCapture("error") {
		t.Error("expected true for error in DebugAll mode")
	}

	// DebugOff 模式
	dm.SetMode(DebugOff, time.Time{})
	if dm.ShouldCapture("error") {
		t.Error("expected false for error in DebugOff mode")
	}
}

// V4.9：TestRateLimit 已迁出至 contrib/debug/ratelimit_sink_test.go
// （限流不再是 DebugManager 内部职责）

func TestCaptureIn(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugAll, sink, time.Time{})

	ctx := context.Background()
	dm.CaptureIn(ctx, "chain1", "rule1", "node1", "condition", `{"temp":25.5}`)

	if sink.Len() != 1 {
		t.Fatalf("expected 1 event, got %d", sink.Len())
	}
	ev := sink.events[0]
	if ev.EventType != EventIn {
		t.Errorf("expected EventIn, got %s", ev.EventType)
	}
	if ev.ChainID != "chain1" {
		t.Errorf("expected chain1, got %s", ev.ChainID)
	}
	if ev.RuleID != "rule1" {
		t.Errorf("expected rule1, got %s", ev.RuleID)
	}
	if ev.NodeID != "node1" {
		t.Errorf("expected node1, got %s", ev.NodeID)
	}
	if ev.NodeType != "condition" {
		t.Errorf("expected condition, got %s", ev.NodeType)
	}
	if ev.RelationType != "in" {
		t.Errorf("expected in, got %s", ev.RelationType)
	}
	if ev.DataSnapshot != `{"temp":25.5}` {
		t.Errorf("expected data snapshot, got %s", ev.DataSnapshot)
	}
	if ev.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
}

func TestCaptureOut(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugAll, sink, time.Time{})

	ctx := context.Background()
	dm.CaptureOut(ctx, "chain1", "rule1", "node1", "action:alarm", "matched", `{"level":"high"}`, 1500, "")

	if sink.Len() != 1 {
		t.Fatalf("expected 1 event, got %d", sink.Len())
	}
	ev := sink.events[0]
	if ev.EventType != EventOut {
		t.Errorf("expected EventOut, got %s", ev.EventType)
	}
	if ev.RelationType != "matched" {
		t.Errorf("expected matched, got %s", ev.RelationType)
	}
	if ev.NodeType != "action:alarm" {
		t.Errorf("expected action:alarm, got %s", ev.NodeType)
	}
	if ev.DurationNs != 1500 {
		t.Errorf("expected 1500, got %d", ev.DurationNs)
	}
}

func TestCaptureOutWithError(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugFailures, sink, time.Time{})

	ctx := context.Background()
	dm.CaptureOut(ctx, "chain1", "rule1", "node1", "action:alarm", "error", `{}`, 2000, "timeout error")

	if sink.Len() != 1 {
		t.Fatalf("expected 1 event, got %d", sink.Len())
	}
	ev := sink.events[0]
	if ev.Error != "timeout error" {
		t.Errorf("expected 'timeout error', got %s", ev.Error)
	}
	if ev.DurationNs != 2000 {
		t.Errorf("expected 2000, got %d", ev.DurationNs)
	}
}

func TestCaptureDisabledWhenSinkSetToNil(t *testing.T) {
	dm := NewDebugManager(DebugAll, nil, time.Time{})
	// 初始时 sink 为 nil，enabled 为 false
	if dm.Enabled() {
		t.Error("expected disabled when sink is nil")
	}

	// 设置有效的 sink
	sink := &mockSink{}
	dm.SetSink(sink)
	if !dm.Enabled() {
		t.Error("expected enabled after setting sink")
	}

	// 再设为 nil
	dm.SetSink(nil)
	if dm.Enabled() {
		t.Error("expected disabled after setting sink to nil")
	}
}

func TestNoopSink(t *testing.T) {
	sink := NoopSink()
	ctx := context.Background()
	if err := sink.WriteEvent(ctx, DebugEvent{}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConcurrentCapture(t *testing.T) {
	sink := &mockSink{}
	dm := NewDebugManager(DebugAll, sink, time.Time{})

	ctx := context.Background()
	done := make(chan struct{})
	const goroutines = 10
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				dm.CaptureIn(ctx, "chain1", "rule1", "node1", "condition", `{"v":1}`)
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	expected := goroutines * iterations
	if sink.Len() != expected {
		t.Errorf("expected %d events, got %d", expected, sink.Len())
	}
}

func TestLogSinkWriteEvent(t *testing.T) {
	// V3.7: LogSink 已迁至 contrib/slog（重命名为 DebugLogSink），原测试随之迁移。
	t.Skip("V3.7: LogSink moved to contrib/slog.DebugLogSink; see contrib/slog/debug_sink_test.go")
}
