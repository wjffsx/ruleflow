package adapter

import (
	"context"
	"testing"
)

// ─────────────────────────────────────────────
//  Mock 数据点
// ─────────────────────────────────────────────

type mockDataPoint struct {
	deviceID      string
	pointName     string
	pointType     string
	value         float64
	quality       int
	upperLimit    float64
	upperValid    bool
	lowerLimit    float64
	lowerValid    bool
	limitExceeded bool
	timestamp     int64
	dropped       bool
	group         string
}

func (m *mockDataPoint) GetDeviceID() string            { return m.deviceID }
func (m *mockDataPoint) GetPointName() string           { return m.pointName }
func (m *mockDataPoint) GetPointType() string           { return m.pointType }
func (m *mockDataPoint) GetValue() float64              { return m.value }
func (m *mockDataPoint) SetValue(v float64)             { m.value = v }
func (m *mockDataPoint) GetQuality() int                { return m.quality }
func (m *mockDataPoint) SetQuality(q int)               { m.quality = q }
func (m *mockDataPoint) GetUpperLimit() (float64, bool) { return m.upperLimit, m.upperValid }
func (m *mockDataPoint) GetLowerLimit() (float64, bool) { return m.lowerLimit, m.lowerValid }
func (m *mockDataPoint) IsLimitExceeded() bool          { return m.limitExceeded }
func (m *mockDataPoint) SetLimitExceeded(v bool)        { m.limitExceeded = v }
func (m *mockDataPoint) GetTimestamp() int64            { return m.timestamp }
func (m *mockDataPoint) IsDropped() bool                { return m.dropped }
func (m *mockDataPoint) SetDropped(v bool)              { m.dropped = v }
func (m *mockDataPoint) GetGroup() string               { return m.group }
func (m *mockDataPoint) PreviousValue() (float64, bool) { return 0, false }
func (m *mockDataPoint) SetPreviousValue(v float64)     {}

// ─────────────────────────────────────────────
//  DataContextAdapter 测试
// ─────────────────────────────────────────────

func TestDataContextAdapter_Delegates(t *testing.T) {
	point := &mockDataPoint{
		deviceID:   "dev_001",
		pointName:  "voltage_a",
		pointType:  "analog",
		value:      220.5,
		quality:    192,
		upperLimit: 250,
		upperValid: true,
		lowerLimit: 200,
		lowerValid: true,
		timestamp:  1717200000,
		group:      "electrical",
	}

	ctx := AcquireDataContext(point)
	defer ReleaseDataContext(ctx)

	// 测试读委托
	if ctx.DeviceID() != "dev_001" {
		t.Errorf("DeviceID = %q, want dev_001", ctx.DeviceID())
	}
	if ctx.PointName() != "voltage_a" {
		t.Errorf("PointName = %q, want voltage_a", ctx.PointName())
	}
	if ctx.PointType() != "analog" {
		t.Errorf("PointType = %q, want analog", ctx.PointType())
	}
	if ctx.Value() != 220.5 {
		t.Errorf("Value = %v, want 220.5", ctx.Value())
	}
	if ctx.Quality() != 192 {
		t.Errorf("Quality = %v, want 192", ctx.Quality())
	}
	if ul, ok := ctx.UpperLimit(); !ok || ul != 250 {
		t.Errorf("UpperLimit = (%v, %v), want (250, true)", ul, ok)
	}
	if ll, ok := ctx.LowerLimit(); !ok || ll != 200 {
		t.Errorf("LowerLimit = (%v, %v), want (200, true)", ll, ok)
	}
	if ctx.Timestamp() != 1717200000 {
		t.Errorf("Timestamp = %v, want 1717200000", ctx.Timestamp())
	}
	if ctx.Raw() != point {
		t.Error("Raw should return original point")
	}

	// 测试写委托
	ctx.SetValue(230.0)
	if point.value != 230.0 {
		t.Errorf("SetValue did not propagate: got %v", point.value)
	}
	ctx.SetQuality(0)
	if point.quality != 0 {
		t.Errorf("SetQuality did not propagate: got %v", point.quality)
	}
	ctx.SetLimitExceeded(true)
	if !point.limitExceeded {
		t.Error("SetLimitExceeded did not propagate")
	}
	ctx.SetDropped(true)
	if !point.dropped {
		t.Error("SetDropped did not propagate")
	}
}

func TestDataContextAdapter_TagsAndTargets(t *testing.T) {
	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a", pointType: "analog"}
	ctx := AcquireDataContext(point)
	defer ReleaseDataContext(ctx)

	// Tags — 延迟分配
	if ctx.GetTag("foo") != "" {
		t.Error("GetTag on nil map should return empty string")
	}
	ctx.SetTag("key1", "val1")
	if ctx.GetTag("key1") != "val1" {
		t.Errorf("GetTag = %q, want val1", ctx.GetTag("key1"))
	}

	// Targets — 预分配
	if ctx.TargetCount() != 0 {
		t.Error("TargetCount should be 0 initially")
	}
	ctx.AddTarget("alarm_system")
	ctx.AddTarget("storage")
	if ctx.TargetCount() != 2 {
		t.Errorf("TargetCount = %v, want 2", ctx.TargetCount())
	}
	if ctx.TargetAt(0) != "alarm_system" {
		t.Errorf("TargetAt(0) = %q, want alarm_system", ctx.TargetAt(0))
	}
	if ctx.TargetAt(1) != "storage" {
		t.Errorf("TargetAt(1) = %q, want storage", ctx.TargetAt(1))
	}
	if ctx.TargetAt(-1) != "" || ctx.TargetAt(99) != "" {
		t.Error("TargetAt out of bounds should return empty string")
	}
}

func TestDataContextAdapter_Pool(t *testing.T) {
	point := &mockDataPoint{deviceID: "dev_001"}
	ctx1 := AcquireDataContext(point)
	ctx1.AddTarget("t1")
	ctx1.SetTag("k1", "v1")
	ReleaseDataContext(ctx1)

	ctx2 := AcquireDataContext(point)
	// 确保复用后状态干净
	if ctx2.TargetCount() != 0 {
		t.Errorf("TargetCount after pool reuse = %v, want 0", ctx2.TargetCount())
	}
	if ctx2.GetTag("k1") != "" {
		t.Error("Tags should be nil after pool release")
	}
	ReleaseDataContext(ctx2)
}

// ─────────────────────────────────────────────
//  背压适配器测试
// ─────────────────────────────────────────────

type mockBPMgr struct {
	shouldAccept bool
	level        int
}

func (m *mockBPMgr) ShouldAccept(pt string) bool { return m.shouldAccept }
func (m *mockBPMgr) CurrentWorstLevel() int      { return m.level }

func TestBackpressureAdapter(t *testing.T) {
	mgr := &mockBPMgr{shouldAccept: true, level: 0}
	adapter := NewBackpressureAdapter(mgr, "analog")

	if !adapter.ShouldAccept("any") {
		t.Error("ShouldAccept should delegate to mgr")
	}
	if adapter.CurrentLevel() != 0 {
		t.Errorf("CurrentLevel = %v, want 0 (Normal)", adapter.CurrentLevel())
	}

	// 测试降级
	mgr.level = 1
	if adapter.CurrentLevel() != 1 {
		t.Errorf("CurrentLevel = %v, want 1 (Degraded)", adapter.CurrentLevel())
	}

	mgr.level = 3
	if adapter.CurrentLevel() != 3 {
		t.Errorf("CurrentLevel = %v, want 3 (Dropping)", adapter.CurrentLevel())
	}
}

// ─────────────────────────────────────────────
//  DLQ 适配器测试
// ─────────────────────────────────────────────

type mockDLQ struct {
	drops  []string
	errors []string
}

func (m *mockDLQ) RecordDrop(deviceID, pointName string, timestamp int64, reason string) {
	m.drops = append(m.drops, deviceID+":"+pointName+":"+reason)
}

func (m *mockDLQ) RecordError(deviceID, pointName string, timestamp int64, errMsg string) {
	m.errors = append(m.errors, deviceID+":"+pointName+":"+errMsg)
}

func TestDLQAdapter(t *testing.T) {
	dlq := &mockDLQ{}
	adapter := NewDLQAdapter(dlq)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a", timestamp: 123}
	ctx := AcquireDataContext(point)
	defer ReleaseDataContext(ctx)

	adapter.TrackDrop(context.Background(), ctx, "rule_1", "limit_exceeded")
	if len(dlq.drops) != 1 || dlq.drops[0] != "dev_001:voltage_a:limit_exceeded" {
		t.Errorf("TrackDrop drops = %v, want [dev_001:voltage_a:limit_exceeded]", dlq.drops)
	}

	adapter.TrackError(context.Background(), ctx, "rule_2", context.DeadlineExceeded)
	if len(dlq.errors) != 1 {
		t.Errorf("TrackError errors = %v, want 1 entry", dlq.errors)
	}
}
