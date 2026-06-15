package stateful

import (
	"context"
	"testing"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// mockDataContext for testing
type mockDataContext struct {
	deviceID      string
	pointName     string
	value         float64
	quality       int
	prevValue     float64
	hasPrev       bool
	timestamp     int64
	tags          map[string]string
	upperLimit    float64
	hasUpper      bool
	lowerLimit    float64
	hasLower      bool
	limitExceeded bool
	dropped       bool
	targets       []string
	spanContext   contract.SpanContext
}

func (m *mockDataContext) DeviceID() string              { return m.deviceID }
func (m *mockDataContext) PointName() string             { return m.pointName }
func (m *mockDataContext) SetPointName(_ string)         {}
func (m *mockDataContext) PointType() string             { return "analog" }
func (m *mockDataContext) FQN() string                   { return m.deviceID + "/" + m.pointName }
func (m *mockDataContext) Value() float64                { return m.value }
func (m *mockDataContext) SetValue(v float64)            { m.value = v }
func (m *mockDataContext) Quality() int                  { return m.quality }
func (m *mockDataContext) SetQuality(q int)              { m.quality = q }
func (m *mockDataContext) UpperLimit() (float64, bool)   { return m.upperLimit, m.hasUpper }
func (m *mockDataContext) LowerLimit() (float64, bool)   { return m.lowerLimit, m.hasLower }
func (m *mockDataContext) LimitExceeded() bool           { return m.limitExceeded }
func (m *mockDataContext) SetLimitExceeded(v bool)       { m.limitExceeded = v }
func (m *mockDataContext) PreviousValue() (float64, bool) { return m.prevValue, m.hasPrev }
func (m *mockDataContext) SetPreviousValue(v float64)    { m.prevValue = v; m.hasPrev = true }
func (m *mockDataContext) GetTag(key string) string      { return m.tags[key] }
func (m *mockDataContext) SetTag(key, value string)      { m.tags[key] = value }
func (m *mockDataContext) TargetCount() int              { return len(m.targets) }
func (m *mockDataContext) TargetAt(i int) string         { return m.targets[i] }
func (m *mockDataContext) AddTarget(target string)       { m.targets = append(m.targets, target) }
func (m *mockDataContext) Dropped() bool                 { return m.dropped }
func (m *mockDataContext) SetDropped(v bool)             { m.dropped = v }
func (m *mockDataContext) Timestamp() int64              { return m.timestamp }
func (m *mockDataContext) SpanContext() contract.SpanContext { return m.spanContext }
func (m *mockDataContext) SetSpanContext(sc contract.SpanContext) { m.spanContext = sc }
func (m *mockDataContext) Raw() any                      { return m }

// mockStatefulDataContext for DurationCondition testing
type mockStatefulDataContext struct {
	mockDataContext
	stateStore core.StateStore
}

func (m *mockStatefulDataContext) StateStore() core.StateStore { return m.stateStore }

// mockStateStore for testing
type mockStateStore struct {
	data map[string]any
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{data: make(map[string]any)}
}

func (s *mockStateStore) Get(key string) (any, bool) {
	v, ok := s.data[key]
	return v, ok
}

func (s *mockStateStore) Set(key string, value any) {
	s.data[key] = value
}

func (s *mockStateStore) Delete(key string) {
	delete(s.data, key)
}

// mockCondition for inner condition testing
type mockCondition struct {
	id      string
	result  bool
}

func (m *mockCondition) ID() string                          { return m.id }
func (m *mockCondition) Type() string                        { return "mock" }
func (m *mockCondition) Description() string                 { return "mock condition" }
func (m *mockCondition) Evaluate(_ context.Context, _ core.DataContext) bool { return m.result }

// ─────────────────────────────────────────────
//  StateChangeCondition Tests
// ─────────────────────────────────────────────

func TestStateChangeCondition_New(t *testing.T) {
	from := 10.0
	to := 20.0
	cond := NewStateChangeCondition("test_id", &from, &to)

	if cond.ID() != "test_id" {
		t.Errorf("expected id test_id, got %s", cond.ID())
	}
	if cond.Type() != "state_change" {
		t.Errorf("expected type state_change, got %s", cond.Type())
	}
}

func TestStateChangeCondition_Evaluate_NoPrevious(t *testing.T) {
	cond := NewStateChangeCondition("test", nil, nil)
	data := &mockDataContext{
		value:   100.0,
		hasPrev: false,
	}

	result := cond.Evaluate(context.Background(), data)
	if result {
		t.Error("expected false when no previous value")
	}
}

func TestStateChangeCondition_Evaluate_AnyChange(t *testing.T) {
	cond := NewStateChangeCondition("test", nil, nil)

	// Value changed
	data := &mockDataContext{
		value:     100.0,
		prevValue: 90.0,
		hasPrev:   true,
	}
	if !cond.Evaluate(context.Background(), data) {
		t.Error("expected true for any change")
	}

	// Value unchanged
	data.prevValue = 100.0
	if cond.Evaluate(context.Background(), data) {
		t.Error("expected false when value unchanged")
	}
}

func TestStateChangeCondition_Evaluate_FromTo(t *testing.T) {
	from := 10.0
	to := 20.0
	cond := NewStateChangeCondition("test", &from, &to)

	// Correct transition
	data := &mockDataContext{
		value:     20.0,
		prevValue: 10.0,
		hasPrev:   true,
	}
	if !cond.Evaluate(context.Background(), data) {
		t.Error("expected true for correct transition")
	}

	// Wrong from value
	data.prevValue = 15.0
	if cond.Evaluate(context.Background(), data) {
		t.Error("expected false for wrong from value")
	}

	// Wrong to value
	data.prevValue = 10.0
	data.value = 25.0
	if cond.Evaluate(context.Background(), data) {
		t.Error("expected false for wrong to value")
	}
}

// ─────────────────────────────────────────────
//  DurationCondition Tests
// ─────────────────────────────────────────────

func TestDurationCondition_New(t *testing.T) {
	inner := &mockCondition{id: "inner", result: true}
	cond := NewDurationCondition("test", inner, 5*time.Second)

	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}
	if cond.Type() != "duration" {
		t.Errorf("expected type duration, got %s", cond.Type())
	}
	if cond.Duration != 5*time.Second {
		t.Errorf("expected duration 5s, got %v", cond.Duration)
	}
}

func TestDurationCondition_Evaluate_NoStateStore(t *testing.T) {
	inner := &mockCondition{id: "inner", result: true}
	cond := NewDurationCondition("test", inner, 5*time.Second)

	data := &mockDataContext{
		value:     100.0,
		timestamp: time.Now().UnixMilli(),
	}

	result := cond.Evaluate(context.Background(), data)
	if result {
		t.Error("expected false when no state store")
	}
}

func TestDurationCondition_Evaluate_FirstEvaluation(t *testing.T) {
	inner := &mockCondition{id: "inner", result: true}
	cond := NewDurationCondition("test", inner, 5*time.Second)

	store := newMockStateStore()
	data := &mockStatefulDataContext{
		mockDataContext: mockDataContext{
			deviceID:  "device1",
			pointName: "point1",
			value:     100.0,
			timestamp: time.Now().UnixMilli(),
		},
		stateStore: store,
	}

	// First evaluation should return false (starts timer)
	result := cond.Evaluate(context.Background(), data)
	if result {
		t.Error("expected false on first evaluation")
	}

	// State should be stored
	_, ok := store.Get("duration:test:device1/point1")
	if !ok {
		t.Error("expected state to be stored")
	}
}

func TestDurationCondition_Evaluate_DurationMet(t *testing.T) {
	inner := &mockCondition{id: "inner", result: true}
	cond := NewDurationCondition("test", inner, 100*time.Millisecond)

	store := newMockStateStore()
	now := time.Now()
	startTime := now.Add(-200 * time.Millisecond) // Started 200ms ago

	store.Set("duration:test:device1/point1", &startTime)

	data := &mockStatefulDataContext{
		mockDataContext: mockDataContext{
			deviceID:  "device1",
			pointName: "point1",
			value:     100.0,
			timestamp: now.UnixMilli(),
		},
		stateStore: store,
	}

	// Duration should be met
	result := cond.Evaluate(context.Background(), data)
	if !result {
		t.Error("expected true when duration met")
	}
}

func TestDurationCondition_Evaluate_InnerFalse(t *testing.T) {
	inner := &mockCondition{id: "inner", result: false}
	cond := NewDurationCondition("test", inner, 5*time.Second)

	store := newMockStateStore()
	startTime := time.Now().Add(-10 * time.Second)
	store.Set("duration:test:device1/point1", &startTime)

	data := &mockStatefulDataContext{
		mockDataContext: mockDataContext{
			deviceID:  "device1",
			pointName: "point1",
			value:     100.0,
			timestamp: time.Now().UnixMilli(),
		},
		stateStore: store,
	}

	// Inner condition false should reset state
	result := cond.Evaluate(context.Background(), data)
	if result {
		t.Error("expected false when inner condition false")
	}

	// State should be deleted
	_, ok := store.Get("duration:test:device1/point1")
	if ok {
		t.Error("expected state to be deleted when inner false")
	}
}

// ─────────────────────────────────────────────
//  Factory Tests
// ─────────────────────────────────────────────

func TestGetFactories(t *testing.T) {
	factories := GetFactories()

	expectedTypes := []string{
		"state_change",
		"duration",
		"trend",
		"periodic",
		"dynamic_threshold",
		"rate_limit_window",
		"limit_recovery",
	}

	for _, expected := range expectedTypes {
		if factories[expected] == nil {
			t.Errorf("missing factory for type: %s", expected)
		}
	}
}

func TestNewStateChangeCondition_Factory(t *testing.T) {
	factory := GetFactories()["state_change"]

	// Test with from/to config
	cond, err := factory("test", map[string]any{
		"from": 10.0,
		"to":   20.0,
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}

	// Test without config
	cond2, err := factory("test2", nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond2.ID() != "test2" {
		t.Errorf("expected id test2, got %s", cond2.ID())
	}
}

func TestNewDurationCondition_Factory(t *testing.T) {
	factory := GetFactories()["duration"]

	// Test valid duration
	cond, err := factory("test", map[string]any{
		"duration": "5s",
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}

	// Test missing duration
	_, err = factory("test2", nil)
	if err == nil {
		t.Error("expected error for missing duration")
	}

	// Test invalid duration
	_, err = factory("test3", map[string]any{
		"duration": "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestNewTrendCondition_Factory(t *testing.T) {
	factory := GetFactories()["trend"]

	// Test with config
	cond, err := factory("test", map[string]any{
		"direction":  "increasing",
		"window":     "5m",
		"threshold":  0.5,
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}

	// Test defaults
	cond2, err := factory("test2", nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond2.ID() != "test2" {
		t.Errorf("expected id test2, got %s", cond2.ID())
	}
}

func TestNewDynamicThresholdCondition_Factory(t *testing.T) {
	factory := GetFactories()["dynamic_threshold"]

	// Test with source
	cond, err := factory("test", map[string]any{
		"operator": "gt",
		"source":   "dynamic_threshold",
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}

	// Test missing source
	_, err = factory("test2", nil)
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestNewRateLimitWindowCondition_Factory(t *testing.T) {
	factory := GetFactories()["rate_limit_window"]

	// Test valid config
	cond, err := factory("test", map[string]any{
		"rate_threshold": 10.0,
		"window":         "5s",
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}

	// Test missing rate_threshold
	_, err = factory("test2", nil)
	if err == nil {
		t.Error("expected error for missing rate_threshold")
	}

	// Test zero rate_threshold
	_, err = factory("test3", map[string]any{
		"rate_threshold": 0.0,
	})
	if err == nil {
		t.Error("expected error for zero rate_threshold")
	}
}

func TestNewLimitRecoveryCondition_Factory(t *testing.T) {
	factory := GetFactories()["limit_recovery"]

	// Test with config
	cond, err := factory("test", map[string]any{
		"duration":   "10s",
		"hysteresis": 0.5,
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond.ID() != "test" {
		t.Errorf("expected id test, got %s", cond.ID())
	}

	// Test defaults
	cond2, err := factory("test2", nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if cond2.ID() != "test2" {
		t.Errorf("expected id test2, got %s", cond2.ID())
	}
}

func TestNormalizeTimestamp(t *testing.T) {
	// Test millisecond timestamp
	msTs := time.Now().UnixMilli()
	msResult := normalizeTimestamp(msTs)
	if msResult.UnixMilli() != msTs {
		t.Errorf("millisecond normalization failed")
	}

	// Test nanosecond timestamp
	nsTs := time.Now().UnixNano()
	nsResult := normalizeTimestamp(nsTs)
	if nsResult.UnixNano() != nsTs {
		t.Errorf("nanosecond normalization failed")
	}
}