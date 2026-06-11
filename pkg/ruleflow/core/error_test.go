package core

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
//  统一错误处理测试
// ─────────────────────────────────────────────

func TestRuleFlowError_Error_Format(t *testing.T) {
	cause := errors.New("underlying failure")
	e := NewActionError("chain1", "rule1", "action1", cause)

	msg := e.Error()
	if !strings.Contains(msg, "[action]") {
		t.Errorf("expected [action] in message, got: %s", msg)
	}
	if !strings.Contains(msg, "chain1") {
		t.Errorf("expected chain1 in message, got: %s", msg)
	}
	if !strings.Contains(msg, "rule1") {
		t.Errorf("expected rule1 in message, got: %s", msg)
	}
	if !strings.Contains(msg, "action1") {
		t.Errorf("expected action1 in message, got: %s", msg)
	}
	if !strings.Contains(msg, "underlying failure") {
		t.Errorf("expected cause in message, got: %s", msg)
	}
}

func TestRuleFlowError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	e := NewConfigError("chain1", cause)

	if errors.Unwrap(e) != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestRuleFlowError_Is(t *testing.T) {
	tests := []struct {
		name string
		err  *RuleFlowError
		sent error
		want bool
	}{
		{"config matches", NewConfigError("c", nil), ErrConfigInvalid, true},
		{"condition matches", NewConditionError("c", "r", nil), ErrConditionEvalFailed, true},
		{"action matches", NewActionError("c", "r", "a", nil), ErrActionExecFailed, true},
		{"panic matches", NewPanicError("c", "r", "boom"), ErrPanicRecovered, true},
		{"dependency matches", NewDependencyError("c", []string{"a", "b"}), ErrCyclicDependency, true},
		{"shutdown matches", NewShutdownError(nil), ErrEngineShutdown, true},
		{"mismatch", NewConfigError("c", nil), ErrActionExecFailed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.sent); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuleFlowError_Timestamp(t *testing.T) {
	before := time.Now()
	e := NewConfigError("c", nil)
	after := time.Now()

	if e.Timestamp.Before(before) || e.Timestamp.After(after) {
		t.Errorf("timestamp %v not in [%v, %v]", e.Timestamp, before, after)
	}
}

func TestRuleFlowError_Panic_StackTrace(t *testing.T) {
	e := NewPanicError("c", "r", "boom")
	if e.Stack == "" {
		t.Error("panic error should have stack trace")
	}
	if !strings.Contains(e.Stack, "goroutine") {
		t.Errorf("stack should contain goroutine info, got: %s", e.Stack[:min(100, len(e.Stack))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
