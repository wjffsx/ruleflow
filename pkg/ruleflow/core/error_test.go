package core

import (
	"context"
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

func TestRuleFlowError_Error_Minimal(t *testing.T) {
	e := &RuleFlowError{Type: ErrorTypeConfig}
	msg := e.Error()
	if !strings.Contains(msg, "[config]") {
		t.Errorf("expected [config] in message, got: %s", msg)
	}
}

func TestRuleFlowError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	e := NewConfigError("chain1", cause)

	if errors.Unwrap(e) != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestRuleFlowError_Unwrap_NilCause(t *testing.T) {
	e := NewConfigError("chain1", nil)
	if errors.Unwrap(e) != nil {
		t.Error("Unwrap should return nil for nil cause")
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
		{"timeout matches", &RuleFlowError{Type: ErrorTypeTimeout}, context.DeadlineExceeded, true},
		{"resource matches", &RuleFlowError{Type: ErrorTypeResource}, ErrResource, true},
		{"mismatch", NewConfigError("c", nil), ErrActionExecFailed, false},
		{"unknown sentinel", NewConfigError("c", nil), errors.New("unknown"), false},
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

func TestRuleFlowError_Retryable(t *testing.T) {
	// Action error with timeout cause should be retryable
	e := NewActionError("c", "r", "a", context.DeadlineExceeded)
	if !e.Retryable {
		t.Error("timeout error should be retryable")
	}

	// Action error with other cause should not be retryable
	e2 := NewActionError("c", "r", "a", errors.New("other"))
	if e2.Retryable {
		t.Error("other error should not be retryable")
	}
}

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		typ  ErrorType
		want string
	}{
		{ErrorTypeConfig, "config"},
		{ErrorTypeCondition, "condition"},
		{ErrorTypeAction, "action"},
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypePanic, "panic"},
		{ErrorTypeDependency, "dependency"},
		{ErrorTypeResource, "resource"},
		{ErrorTypeShutdown, "shutdown"},
		{ErrorType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorAction_String(t *testing.T) {
	tests := []struct {
		act  ErrorAction
		want string
	}{
		{ErrorActionContinue, "continue"},
		{ErrorActionAbort, "abort"},
		{ErrorActionRetry, "retry"},
		{ErrorActionFallback, "fallback"},
		{ErrorAction(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.act.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	// nil error
	if isRetryable(nil) {
		t.Error("nil should not be retryable")
	}

	// timeout error
	if !isRetryable(context.DeadlineExceeded) {
		t.Error("DeadlineExceeded should be retryable")
	}

	// other error
	if isRetryable(errors.New("other")) {
		t.Error("other error should not be retryable")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────
//  ErrorHandler Tests
// ─────────────────────────────────────────────

func TestContinueOnErrorHandler(t *testing.T) {
	h := ContinueOnErrorHandler()
	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	if action != ErrorActionContinue {
		t.Errorf("expected Continue, got %v", action)
	}
}

func TestAbortOnErrorHandler(t *testing.T) {
	h := AbortOnErrorHandler()
	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	if action != ErrorActionAbort {
		t.Errorf("expected Abort, got %v", action)
	}
}

func TestRetryOnceErrorHandler(t *testing.T) {
	h := RetryOnceErrorHandler()
	err := NewConfigError("test", nil)

	// First attempt - should retry
	ctx1 := HandlerContext{ChainID: "test", Attempt: 1}
	action1, herr1 := h.HandleError(ctx1, err)
	if herr1 != nil {
		t.Errorf("unexpected error: %v", herr1)
	}
	if action1 != ErrorActionRetry {
		t.Errorf("expected Retry on first attempt, got %v", action1)
	}

	// Second attempt - should continue
	ctx2 := HandlerContext{ChainID: "test", Attempt: 2}
	action2, herr2 := h.HandleError(ctx2, err)
	if herr2 != nil {
		t.Errorf("unexpected error: %v", herr2)
	}
	if action2 != ErrorActionContinue {
		t.Errorf("expected Continue on second attempt, got %v", action2)
	}
}

func TestChainedErrorHandler(t *testing.T) {
	// Chain: Continue -> Abort -> Continue
	h := &ChainedErrorHandler{
		Handlers: []ErrorHandler{
			ContinueOnErrorHandler(),
			AbortOnErrorHandler(),
			ContinueOnErrorHandler(),
		},
	}

	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	// Should return first non-Continue action (Abort)
	if action != ErrorActionAbort {
		t.Errorf("expected Abort, got %v", action)
	}
}

func TestChainedErrorHandler_AllContinue(t *testing.T) {
	h := &ChainedErrorHandler{
		Handlers: []ErrorHandler{
			ContinueOnErrorHandler(),
			ContinueOnErrorHandler(),
		},
	}

	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	if action != ErrorActionContinue {
		t.Errorf("expected Continue, got %v", action)
	}
}

func TestMetricsErrorHandler(t *testing.T) {
	callCount := 0
	h := &MetricsErrorHandler{
		Inner: AbortOnErrorHandler(),
		OnError: func(_ HandlerContext, _ *RuleFlowError) {
			callCount++
		},
	}

	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	if action != ErrorActionAbort {
		t.Errorf("expected Abort, got %v", action)
	}
	if callCount != 1 {
		t.Errorf("expected OnError to be called once, got %d", callCount)
	}
}

func TestMetricsErrorHandler_NoInner(t *testing.T) {
	h := &MetricsErrorHandler{
		Inner: nil,
	}

	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	if action != ErrorActionContinue {
		t.Errorf("expected Continue with nil Inner, got %v", action)
	}
}

func TestMetricsErrorHandler_NoOnError(t *testing.T) {
	h := &MetricsErrorHandler{
		Inner: AbortOnErrorHandler(),
		OnError: nil,
	}

	ctx := HandlerContext{ChainID: "test"}
	err := NewConfigError("test", nil)

	action, herr := h.HandleError(ctx, err)
	if herr != nil {
		t.Errorf("unexpected error: %v", herr)
	}
	if action != ErrorActionAbort {
		t.Errorf("expected Abort, got %v", action)
	}
}
