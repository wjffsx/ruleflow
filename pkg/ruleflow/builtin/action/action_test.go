package action

import (
	"context"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  测试用 mock DataContext
// ─────────────────────────────────────────────

type mockData struct {
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

func newMock() *mockData {
	return &mockData{
		deviceID:  "device-001",
		pointName: "voltage",
		pointType: "analog",
		value:     220.5,
		quality:   192,
		tags:      make(map[string]string),
	}
}

func (m *mockData) DeviceID() string                       { return m.deviceID }
func (m *mockData) PointName() string                      { return m.pointName }
func (m *mockData) PointType() string                      { return m.pointType }
func (m *mockData) FQN() string                            { return m.deviceID + "/" + m.pointName }
func (m *mockData) Value() float64                         { return m.value }
func (m *mockData) SetValue(v float64)                     { m.value = v }
func (m *mockData) Quality() int                           { return m.quality }
func (m *mockData) SetQuality(q int)                       { m.quality = q }
func (m *mockData) UpperLimit() (float64, bool)            { return m.upperLimit, m.hasUpper }
func (m *mockData) LowerLimit() (float64, bool)            { return m.lowerLimit, m.hasLower }
func (m *mockData) LimitExceeded() bool                    { return m.limitExceeded }
func (m *mockData) SetLimitExceeded(v bool)                { m.limitExceeded = v }
func (m *mockData) GetTag(key string) string               { return m.tags[key] }
func (m *mockData) SetTag(key, value string)               { m.tags[key] = value }
func (m *mockData) TargetCount() int                       { return len(m.targets) }
func (m *mockData) TargetAt(i int) string                  { return m.targets[i] }
func (m *mockData) AddTarget(target string)                { m.targets = append(m.targets, target) }
func (m *mockData) Dropped() bool                          { return m.dropped }
func (m *mockData) SetDropped(v bool)                      { m.dropped = v }
func (m *mockData) Timestamp() int64                       { return m.timestamp }
func (m *mockData) SpanContext() contract.SpanContext      { return contract.SpanContext{} }
func (m *mockData) SetSpanContext(sc contract.SpanContext) {}
func (m *mockData) Raw() any                               { return nil }
func (m *mockData) PreviousValue() (float64, bool)         { return 0, false }
func (m *mockData) SetPreviousValue(v float64)             {}

func TestTransformAction_Scale(t *testing.T) {
	scale := 2.0
	a := NewTransformAction("a1", &scale, nil, "")
	data := newMock()
	data.value = 100
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Value() != 200 {
		t.Errorf("expected 200, got %f", data.Value())
	}
}

func TestTransformAction_Offset(t *testing.T) {
	offset := 10.0
	a := NewTransformAction("a1", nil, &offset, "")
	data := newMock()
	data.value = 100
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Value() != 110 {
		t.Errorf("expected 110, got %f", data.Value())
	}
}

func TestTransformAction_ScaleAndOffset(t *testing.T) {
	scale := 2.0
	offset := 5.0
	a := NewTransformAction("a1", &scale, &offset, "kV")
	data := newMock()
	data.value = 100
	a.Execute(context.Background(), data)
	if data.Value() != 205 { // 100*2 + 5
		t.Errorf("expected 205, got %f", data.Value())
	}
}

func TestTransformAction_NoOp(t *testing.T) {
	a := NewTransformAction("a1", nil, nil, "")
	data := newMock()
	data.value = 100
	a.Execute(context.Background(), data)
	if data.Value() != 100 {
		t.Errorf("no-op transform should not change value, got %f", data.Value())
	}
}

// ─────────────────────────────────────────────
//  RenameAction 测试
// ─────────────────────────────────────────────

func TestRenameAction(t *testing.T) {
	a := NewRenameAction("a1", "new_name")
	data := newMock()
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.GetTag("_rename") != "new_name" {
		t.Errorf("expected tag _rename=new_name, got %s", data.GetTag("_rename"))
	}
}

// ─────────────────────────────────────────────
//  TagAction 测试
// ─────────────────────────────────────────────

func TestTagAction(t *testing.T) {
	tags := map[string]string{"source": "ruleflow", "level": "high"}
	a := NewTagAction("a1", tags)
	data := newMock()
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.GetTag("source") != "ruleflow" {
		t.Errorf("expected tag source=ruleflow, got %s", data.GetTag("source"))
	}
	if data.GetTag("level") != "high" {
		t.Errorf("expected tag level=high, got %s", data.GetTag("level"))
	}
}

// ─────────────────────────────────────────────
//  DropAction 测试
// ─────────────────────────────────────────────

func TestDropAction(t *testing.T) {
	a := NewDropAction("a1")
	data := newMock()
	err := a.Execute(context.Background(), data)
	if err != core.ErrDropData {
		t.Errorf("expected ErrDropData, got %v", err)
	}
}

// ─────────────────────────────────────────────
//  RouteAction 测试
// ─────────────────────────────────────────────

func TestRouteAction(t *testing.T) {
	a := NewRouteAction("a1", []string{"kafka", "influxdb"})
	data := newMock()
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.TargetCount() != 2 {
		t.Errorf("expected 2 targets, got %d", data.TargetCount())
	}
	if data.TargetAt(0) != "kafka" || data.TargetAt(1) != "influxdb" {
		t.Errorf("unexpected targets: %v", data.targets)
	}
}

// ─────────────────────────────────────────────
//  LimitCheckAction 测试
// ─────────────────────────────────────────────

func TestLimitCheckAction_WithinLimits(t *testing.T) {
	a := NewLimitCheckAction("a1")
	data := newMock()
	data.value = 50
	data.upperLimit = 100
	data.hasUpper = true
	data.lowerLimit = 0
	data.hasLower = true
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.LimitExceeded() {
		t.Error("should not be exceeded when within limits")
	}
}

func TestLimitCheckAction_ExceedsUpper(t *testing.T) {
	a := NewLimitCheckAction("a1")
	data := newMock()
	data.value = 150
	data.upperLimit = 100
	data.hasUpper = true
	a.Execute(context.Background(), data)
	if !data.LimitExceeded() {
		t.Error("should be exceeded when above upper limit")
	}
}

func TestLimitCheckAction_ExceedsLower(t *testing.T) {
	a := NewLimitCheckAction("a1")
	data := newMock()
	data.value = -10
	data.lowerLimit = 0
	data.hasLower = true
	a.Execute(context.Background(), data)
	if !data.LimitExceeded() {
		t.Error("should be exceeded when below lower limit")
	}
}

func TestLimitCheckAction_NoLimits(t *testing.T) {
	a := NewLimitCheckAction("a1")
	data := newMock()
	data.value = 9999
	a.Execute(context.Background(), data)
	if data.LimitExceeded() {
		t.Error("should not be exceeded when no limits set")
	}
}

// ─────────────────────────────────────────────
//  QualityMarkAction 测试
// ─────────────────────────────────────────────

func TestQualityMarkAction_Good(t *testing.T) {
	a := NewQualityMarkAction("a1", 192, 0)
	data := newMock()
	data.limitExceeded = false
	a.Execute(context.Background(), data)
	if data.Quality() != 192 {
		t.Errorf("expected quality 192 (good), got %d", data.Quality())
	}
}

func TestQualityMarkAction_Bad(t *testing.T) {
	a := NewQualityMarkAction("a1", 192, 0)
	data := newMock()
	data.limitExceeded = true
	a.Execute(context.Background(), data)
	if data.Quality() != 0 {
		t.Errorf("expected quality 0 (bad), got %d", data.Quality())
	}
}

// ─────────────────────────────────────────────
//  AlarmNotifyAction 测试
// ─────────────────────────────────────────────

func TestAlarmNotifyAction(t *testing.T) {
	var notified bool
	a := NewAlarmNotifyAction("a1", "critical", "over limit", func(deviceID, pointName, severity, message string) {
		notified = true
		if deviceID != "device-001" {
			t.Errorf("expected device-001, got %s", deviceID)
		}
		if severity != "critical" {
			t.Errorf("expected critical, got %s", severity)
		}
	})
	data := newMock()
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !notified {
		t.Error("notify func should have been called")
	}
}

func TestAlarmNotifyAction_NilFunc(t *testing.T) {
	a := NewAlarmNotifyAction("a1", "warning", "test", nil)
	data := newMock()
	if err := a.Execute(context.Background(), data); err != nil {
		t.Fatalf("nil notifyFunc should not error: %v", err)
	}
}

// ─────────────────────────────────────────────
//  Action Type/ID/Description 测试
// ─────────────────────────────────────────────

func TestActionMetadata(t *testing.T) {
	tests := []struct {
		action   core.Action
		wantType string
		wantID   string
	}{
		{NewTransformAction("t1", nil, nil, ""), "transform", "t1"},
		{NewRenameAction("r1", "x"), "rename", "r1"},
		{NewTagAction("tag1", nil), "tag", "tag1"},
		{NewDropAction("d1"), "drop", "d1"},
		{NewRouteAction("rt1", nil), "route", "rt1"},
		{NewLimitCheckAction("lc1"), "limit_check", "lc1"},
		{NewQualityMarkAction("qm1", 0, 1), "quality_mark", "qm1"},
		{NewAlarmNotifyAction("an1", "", "", nil), "alarm_notify", "an1"},
	}
	for _, tt := range tests {
		if tt.action.Type() != tt.wantType {
			t.Errorf("expected type %s, got %s", tt.wantType, tt.action.Type())
		}
		if tt.action.ID() != tt.wantID {
			t.Errorf("expected ID %s, got %s", tt.wantID, tt.action.ID())
		}
		if tt.action.Description() == "" {
			t.Error("description should not be empty")
		}
	}
}

// ─────────────────────────────────────────────
//  Condition Metadata 测试（从 condition 包导入）
// ─────────────────────────────────────────────

func TestConditionMetadata(t *testing.T) {
	tests := []struct {
		condition core.Condition
		wantType  string
		wantID    string
	}{
		{condition.NewDeviceTypeCondition("c1", []string{"analog"}), "device_type", "c1"},
		{condition.NewPointNameCondition("c2", []string{"voltage"}), "point_name", "c2"},
		{condition.NewLimitExceededCondition("c3"), "limit_exceeded", "c3"},
	}
	for _, tt := range tests {
		if tt.condition.Type() != tt.wantType {
			t.Errorf("expected type %s, got %s", tt.wantType, tt.condition.Type())
		}
		if tt.condition.ID() != tt.wantID {
			t.Errorf("expected ID %s, got %s", tt.wantID, tt.condition.ID())
		}
		if tt.condition.Description() == "" {
			t.Error("description should not be empty")
		}
	}
}
