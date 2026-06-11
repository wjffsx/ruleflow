package slog

import (
	"context"
	"log/slog"
	"testing"

	"github.com/vpptu/ruleflow/pkg/ruleflow/debug"
)

func TestDebugLogSink_WriteEvent(t *testing.T) {
	var buf mockLogBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sink := NewDebugLogSink(logger)

	ctx := context.Background()
	ev := debug.DebugEvent{
		EventType:    debug.EventIn,
		ChainID:      "chain1",
		RuleID:       "rule1",
		NodeID:       "node1",
		NodeType:     "condition",
		RelationType: "in",
		DataSnapshot: `{"temp":25}`,
		Timestamp:    1234567890,
		DurationNs:   1000,
	}

	if err := sink.WriteEvent(ctx, ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected log output")
	}
	if !contains(output, "chain1") {
		t.Errorf("expected chain1 in log output, got: %s", output)
	}
	if !contains(output, "rule1") {
		t.Errorf("expected rule1 in log output, got: %s", output)
	}
}

// mockLogBuffer 用于捕获 slog 输出
type mockLogBuffer struct {
	data string
}

func (b *mockLogBuffer) Write(p []byte) (n int, err error) {
	b.data += string(p)
	return len(p), nil
}

func (b *mockLogBuffer) String() string { return b.data }

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
