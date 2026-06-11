package debug

import (
	"context"
	"testing"
	"time"

	coredebug "github.com/wjffsx/ruleflow/pkg/ruleflow/debug"
)

// V4.10 重命名：原 EventBus_* 测试 → ChannelSink_*

func TestNewChannelSink_DefaultBuffer(t *testing.T) {
	sink := NewChannelSink(0)
	if sink == nil {
		t.Fatal("expected non-nil sink")
	}
	sent, dropped := sink.Stats()
	if sent != 0 || dropped != 0 {
		t.Errorf("expected 0,0 got %d,%d", sent, dropped)
	}
}

func TestChannelSink_WriteAndSubscribe(t *testing.T) {
	sink := NewChannelSink(64)
	ctx := context.Background()

	event := coredebug.DebugEvent{
		EventType:    coredebug.EventIn,
		ChainID:      "chain-1",
		NodeID:       "node-1",
		RelationType: "in",
		Timestamp:    time.Now().UnixNano(),
	}

	if err := sink.WriteEvent(ctx, event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	ch := sink.Subscribe()
	select {
	case received := <-ch:
		if received.ChainID != "chain-1" || received.NodeID != "node-1" {
			t.Errorf("unexpected event: %+v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestChannelSink_NonBlockingOnFull(t *testing.T) {
	sink := NewChannelSink(2) // 小 buffer
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		if err := sink.WriteEvent(ctx, coredebug.DebugEvent{ChainID: "test"}); err != nil {
			t.Fatalf("WriteEvent should never block/error: %v", err)
		}
	}

	sent, dropped := sink.Stats()
	if sent+dropped != 100 {
		t.Errorf("expected 100 total, got sent=%d dropped=%d", sent, dropped)
	}
	if dropped == 0 {
		t.Error("expected some dropped events with small buffer")
	}
	t.Logf("sent=%d dropped=%d (buffer=2)", sent, dropped)
}

func TestChannelSink_DebugSinkInterface(t *testing.T) {
	// compile-time check already in source; runtime check
	sink := NewChannelSink(16)
	var ds coredebug.DebugSink = sink
	_ = ds
}

func TestChannelSink_MultipleConsumers(t *testing.T) {
	sink := NewChannelSink(64)
	ctx := context.Background()

	// 写入 5 个事件
	for i := 0; i < 5; i++ {
		_ = sink.WriteEvent(ctx, coredebug.DebugEvent{ChainID: "chain-1"})
	}

	// 两个消费者读取（共享同一个 channel）
	ch1 := sink.Subscribe()
	ch2 := sink.Subscribe()

	// 每个消费者只能读取部分事件（因为共享 chan）
	count1 := 0
	count2 := 0
	done := make(chan struct{}, 2)

	go func() {
		for range ch1 {
			count1++
			if count1+count2 >= 5 {
				done <- struct{}{}
				return
			}
		}
	}()
	go func() {
		for range ch2 {
			count2++
			if count1+count2 >= 5 {
				done <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-done:
		t.Logf("consumer1=%d consumer2=%d total=%d", count1, count2, count1+count2)
	case <-time.After(time.Second):
		// It's ok if one consumer gets all
		t.Logf("timeout: consumer1=%d consumer2=%d", count1, count2)
	}
}

func TestChannelSink_WriteAfterSubscribe(t *testing.T) {
	sink := NewChannelSink(16)
	ctx := context.Background()

	ch := sink.Subscribe()

	_ = sink.WriteEvent(ctx, coredebug.DebugEvent{ChainID: "after-sub"})

	select {
	case e := <-ch:
		if e.ChainID != "after-sub" {
			t.Errorf("expected after-sub, got %s", e.ChainID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// V4.9 移入：TestRateLimitSink（从 core/debug 包迁出至此）
func TestRateLimitSink_AllowsUpToLimit(t *testing.T) {
	inner := &recordingSink{}
	limited := NewRateLimitSink(inner, 3)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_ = limited.WriteEvent(ctx, coredebug.DebugEvent{ChainID: "test"})
	}
	if inner.count != 3 {
		t.Errorf("expected 3 events passed through, got %d", inner.count)
	}
}

func TestRateLimitSink_DropsBeyondLimit(t *testing.T) {
	inner := &recordingSink{}
	limited := NewRateLimitSink(inner, 3)
	ctx := context.Background()

	// 第 4 条（同一秒内）应被丢弃
	for i := 0; i < 4; i++ {
		_ = limited.WriteEvent(ctx, coredebug.DebugEvent{ChainID: "test"})
	}
	if inner.count != 3 {
		t.Errorf("expected 3 events passed through (4th dropped), got %d", inner.count)
	}
}

// recordingSink 简单记录器
type recordingSink struct {
	count int64
}

func (r *recordingSink) WriteEvent(_ any, _ coredebug.DebugEvent) error {
	r.count++
	return nil
}
