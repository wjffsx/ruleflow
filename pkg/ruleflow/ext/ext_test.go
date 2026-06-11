package ext

import (
	"context"
	"testing"

	"github.com/vpptu/ruleflow/pkg/ruleflow/adapter"
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
//  AlarmNotifyExtAction 测试
// ─────────────────────────────────────────────

type mockAlarmStore struct {
	events []any
}

func (m *mockAlarmStore) SaveEvent(_ context.Context, event any) error {
	m.events = append(m.events, event)
	return nil
}

func TestAlarmNotifyExtAction(t *testing.T) {
	store := &mockAlarmStore{}
	action := NewAlarmNotifyExtAction("a1", "threshold", "critical", store)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a", value: 350, timestamp: 1717200000}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	if ctx.GetTag("_alarm_notified") != "true" {
		t.Error("expected alarm_notified tag")
	}
	if ctx.GetTag("_alarm_type") != "threshold" {
		t.Errorf("alarm_type = %q, want threshold", ctx.GetTag("_alarm_type"))
	}
	if ctx.GetTag("_alarm_severity") != "critical" {
		t.Errorf("severity = %q, want critical", ctx.GetTag("_alarm_severity"))
	}

	// 确认异步存储触发
	if len(store.events) != 1 {
		t.Errorf("alarm events = %d, want 1", len(store.events))
	}
}

func TestAlarmNotifyExtAction_DefaultSeverity(t *testing.T) {
	action := NewAlarmNotifyExtAction("a2", "", "", nil)
	point := &mockDataPoint{deviceID: "dev_002", pointName: "current_a"}
	ctx := adapter.AcquireDataContext(point)

	_ = action.Execute(context.Background(), ctx)
	if ctx.GetTag("_alarm_type") != "threshold" {
		t.Errorf("default alarm_type = %q, want threshold", ctx.GetTag("_alarm_type"))
	}
	if ctx.GetTag("_alarm_severity") != "warning" {
		t.Errorf("default severity = %q, want warning", ctx.GetTag("_alarm_severity"))
	}
}

func TestAlarmNotifyExtAction_TagOverride(t *testing.T) {
	action := NewAlarmNotifyExtAction("a3", "threshold", "info", nil)
	point := &mockDataPoint{deviceID: "dev_003", pointName: "frequency"}
	ctx := adapter.AcquireDataContext(point)
	ctx.SetTag("alarm_type", "over_freq")
	ctx.SetTag("alarm_severity", "critical")

	_ = action.Execute(context.Background(), ctx)
	if ctx.GetTag("_alarm_type") != "over_freq" {
		t.Errorf("overridden alarm_type = %q, want over_freq", ctx.GetTag("_alarm_type"))
	}
	if ctx.GetTag("_alarm_severity") != "critical" {
		t.Errorf("overridden severity = %q, want critical", ctx.GetTag("_alarm_severity"))
	}
}

// ─────────────────────────────────────────────
//  QualityMarkExtAction 测试
// ─────────────────────────────────────────────

type mockQualityStore struct {
	calls []string
}

func (m *mockQualityStore) UpdateDataQuality(deviceID, pointName string, quality int) error {
	m.calls = append(m.calls, deviceID+":"+pointName)
	return nil
}

func TestQualityMarkExtAction(t *testing.T) {
	store := &mockQualityStore{}
	action := NewQualityMarkExtAction("q1", "GOOD", store)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a"}
	ctx := adapter.AcquireDataContext(point)

	_ = action.Execute(context.Background(), ctx)
	if ctx.Quality() != 1 {
		t.Errorf("quality = %d, want 1 (GOOD)", ctx.Quality())
	}
	if ctx.GetTag("_vppt_quality") != "GOOD" {
		t.Errorf("tag = %q, want GOOD", ctx.GetTag("_vppt_quality"))
	}
}

func TestQualityMarkExtAction_ExplicitBad(t *testing.T) {
	action := NewQualityMarkExtAction("q2", "BAD", nil)
	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a"}
	ctx := adapter.AcquireDataContext(point)

	_ = action.Execute(context.Background(), ctx)
	if ctx.Quality() != 0 {
		t.Errorf("quality = %d, want 0 (BAD)", ctx.Quality())
	}
}

func TestQualityMarkExtAction_TagOverride(t *testing.T) {
	action := NewQualityMarkExtAction("q3", "GOOD", nil)
	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a"}
	ctx := adapter.AcquireDataContext(point)
	ctx.SetTag("quality", "BAD")

	_ = action.Execute(context.Background(), ctx)
	if ctx.Quality() != 0 {
		t.Errorf("quality = %d, want 0 (BAD override)", ctx.Quality())
	}
}

func TestQualityMarkExtAction_DefaultEmpty(t *testing.T) {
	action := NewQualityMarkExtAction("q4", "", nil)
	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a"}
	ctx := adapter.AcquireDataContext(point)

	_ = action.Execute(context.Background(), ctx)
	if ctx.Quality() != 1 {
		t.Errorf("quality = %d, want 1 (default GOOD)", ctx.Quality())
	}
}

// ─────────────────────────────────────────────
//  CalcNodeAction 测试
// ─────────────────────────────────────────────

func TestCalcNodeAction(t *testing.T) {
	action := NewCalcNodeAction("c1", "value * 2 + 1", []string{"value"}, "calc_output")
	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a", value: 220}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	if ctx.Value() != 441 {
		t.Errorf("calculated value = %v, want 441", ctx.Value())
	}
	if ctx.GetTag("calc_output") != "calc_output" {
		t.Errorf("calc_output tag = %q", ctx.GetTag("calc_output"))
	}
}

func TestCalcNodeAction_Func(t *testing.T) {
	action := NewCalcNodeAction("c3", "value * 2 + abs(quality - 1)", []string{}, "output")
	point := &mockDataPoint{deviceID: "dev_001", pointName: "test", value: 10, quality: 0}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}
	if ctx.Value() != 21 {
		t.Errorf("10*2+|0-1| = %v, want 21", ctx.Value())
	}
}

// ─────────────────────────────────────────────
//  StorageWriteAction 测试
// ─────────────────────────────────────────────

type mockStorage struct {
	dataPoints []any
}

func (m *mockStorage) WriteData(dp any) error {
	m.dataPoints = append(m.dataPoints, dp)
	return nil
}
func (m *mockStorage) WriteDataBatch(dps []any) error {
	m.dataPoints = append(m.dataPoints, dps...)
	return nil
}

func TestStorageWriteAction(t *testing.T) {
	store := &mockStorage{}
	action := NewStorageWriteAction("s1", "realtime", store, nil)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a", value: 220.5, timestamp: 1000, quality: 1}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	if len(store.dataPoints) != 1 {
		t.Fatalf("data points = %d, want 1", len(store.dataPoints))
	}
	dp := store.dataPoints[0].(map[string]any)
	if dp["device_id"] != "dev_001" {
		t.Errorf("device_id = %v", dp["device_id"])
	}
	if dp["value"] != 220.5 {
		t.Errorf("value = %v, want 220.5", dp["value"])
	}
}

func TestStorageWriteAction_Target(t *testing.T) {
	action := NewStorageWriteAction("s2", "alarm_system", nil, nil)
	point := &mockDataPoint{deviceID: "dev_001", pointName: "current_a", timestamp: 1000}
	ctx := adapter.AcquireDataContext(point)

	_ = action.Execute(context.Background(), ctx)
	if ctx.TargetCount() != 1 || ctx.TargetAt(0) != "alarm_system" {
		t.Errorf("target = %q, want alarm_system", ctx.TargetAt(0))
	}
}

func TestStorageWriteAction_EmptyDevice(t *testing.T) {
	action := NewStorageWriteAction("s3", "", nil, nil)
	point := &mockDataPoint{deviceID: "", pointName: ""}
	ctx := adapter.AcquireDataContext(point)
	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("empty device should not error: %v", err)
	}
}

// ─────────────────────────────────────────────
//  AggregationWriteAction 测试
// ─────────────────────────────────────────────

func TestAggregationWriteAction(t *testing.T) {
	store := &mockStorage{}
	action := NewAggregationWriteAction("ag1", store)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "total_power", value: 500, group: "plant_a"}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	if len(store.dataPoints) == 0 {
		t.Fatal("expected aggregated data points")
	}
	dp := store.dataPoints[0].(map[string]any)
	if dp["device_id"] != "plant_a" {
		t.Errorf("aggregated device_id = %v, want plant_a", dp["device_id"])
	}
}

func TestAggregationWriteAction_NoStore(t *testing.T) {
	action := NewAggregationWriteAction("ag2", nil)
	point := &mockDataPoint{deviceID: "dev_001", pointName: "test", group: "plant_a"}
	ctx := adapter.AcquireDataContext(point)
	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("no store should not error: %v", err)
	}
}

// ─────────────────────────────────────────────
//  DeviceAggregateAction 测试
// ─────────────────────────────────────────────

type mockBatchReader struct {
	data []any
}

func (m *mockBatchReader) GetRealtimeDataBatchMulti(_ []string, _ []string) ([]any, error) {
	return m.data, nil
}
func (m *mockBatchReader) GetAllRealtimeData() ([]any, error) {
	return m.data, nil
}

func TestDeviceAggregateAction(t *testing.T) {
	reader := &mockBatchReader{
		data: []any{
			map[string]any{"device_id": "dev_001", "point_name": "power", "value": 100.0},
			map[string]any{"device_id": "dev_002", "point_name": "power", "value": 200.0},
		},
	}
	mappings := []OutputMapping{
		{Category: "*", Output: "total_power", Target: "plant_agg"},
	}
	catProvider := &mockCatProvider{cat: "plant"}
	action := NewDeviceAggregateAction("da1", "power", mappings, catProvider, reader)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "power"}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	total := ctx.GetTag("total_power")
	if total != "300.00" {
		t.Errorf("aggregated total_power = %q, want 300.00", total)
	}
	target := ctx.GetTag("total_power_target")
	if target != "plant_agg" {
		t.Errorf("target = %q, want plant_agg", target)
	}
}

type mockCatProvider struct {
	cat string
}

func (m *mockCatProvider) GetDeviceCategory(_ string) string { return m.cat }

// ─────────────────────────────────────────────
//  StatusChangeLogAction 测试
// ─────────────────────────────────────────────

type mockEventStore struct {
	events []any
}

func (m *mockEventStore) Save(_ context.Context, event any) error {
	m.events = append(m.events, event)
	return nil
}

func TestStatusChangeLogAction(t *testing.T) {
	store := &mockEventStore{}
	action := NewStatusChangeLogAction("sc1", store, nil)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a", timestamp: 1000}
	ctx := adapter.AcquireDataContext(point)
	ctx.SetTag("changeLogEnabled", "true")
	ctx.SetTag("old_value", "220")
	ctx.SetTag("new_value", "230")

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	if len(store.events) != 1 {
		t.Errorf("events = %d, want 1", len(store.events))
	}
}

func TestStatusChangeLogAction_Disabled(t *testing.T) {
	store := &mockEventStore{}
	action := NewStatusChangeLogAction("sc2", store, nil)

	point := &mockDataPoint{deviceID: "dev_001", pointName: "voltage_a"}
	ctx := adapter.AcquireDataContext(point)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}

	if len(store.events) != 0 {
		t.Error("expected no events when changeLogEnabled != true")
	}
}

// ─────────────────────────────────────────────
//  ExprFilterCondition 测试
// ─────────────────────────────────────────────

func TestExprFilterCondition_Basic(t *testing.T) {
	c := NewExprFilterCondition("e1", "value > 100")
	point := &mockDataPoint{value: 150}
	ctx := adapter.AcquireDataContext(point)

	if !c.Evaluate(context.Background(), ctx) {
		t.Error("150 > 100 should be true")
	}

	point.value = 50
	if c.Evaluate(context.Background(), ctx) {
		t.Error("50 > 100 should be false")
	}
}

func TestExprFilterCondition_EmptyExpression(t *testing.T) {
	c := NewExprFilterCondition("e2", "")
	point := &mockDataPoint{value: 100}
	ctx := adapter.AcquireDataContext(point)

	if !c.Evaluate(context.Background(), ctx) {
		t.Error("empty expression should return true")
	}
}

// ─────────────────────────────────────────────
//  ExprSwitchAction 测试
// ─────────────────────────────────────────────

func TestExprSwitchAction_Basic(t *testing.T) {
	a := NewExprSwitchAction("es1", "value > 100")

	point := &mockDataPoint{value: 200}
	ctx := adapter.AcquireDataContext(point)

	err := a.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}
	if ctx.GetTag("_switch_result") != "true" {
		t.Errorf("switch result = %q, want true", ctx.GetTag("_switch_result"))
	}

	point2 := &mockDataPoint{value: 50}
	ctx2 := adapter.AcquireDataContext(point2)
	_ = a.Execute(context.Background(), ctx2)
	if ctx2.GetTag("_switch_result") != "false" {
		t.Errorf("switch result = %q, want false", ctx2.GetTag("_switch_result"))
	}
}

func TestExprSwitchAction_EmptyExpression(t *testing.T) {
	a := NewExprSwitchAction("es2", "")
	point := &mockDataPoint{value: 100}
	ctx := adapter.AcquireDataContext(point)

	err := a.Execute(context.Background(), ctx)
	if err == nil {
		t.Error("expected error for empty expression")
	}
}

// ─────────────────────────────────────────────
//  Package (NodePackage) 测试
// ─────────────────────────────────────────────

func TestPackage_ActionFactories(t *testing.T) {
	factories := Package.GetActionFactories()
	expected := []string{
		"alarm_notify_ext", "quality_mark_ext", "calc_node",
		"storage_write", "aggregation_write", "device_aggregator",
		"status_change_log", "expr_switch",
		"multi_device_control", "strategy_execute",
	}
	for _, name := range expected {
		if _, ok := factories[name]; !ok {
			t.Errorf("missing action factory: %s", name)
		}
	}
}

func TestPackage_ConditionFactories(t *testing.T) {
	factories := Package.GetConditionFactories()
	if _, ok := factories["expr_filter"]; !ok {
		t.Error("missing expr_filter condition factory")
	}
	if _, ok := factories["historical_compare"]; !ok {
		t.Error("missing historical_compare condition factory")
	}
}

func TestActionMetaList(t *testing.T) {
	metas := ActionMetaList()
	if len(metas) != 10 {
		t.Errorf("component metas = %d, want 10", len(metas))
	}
}
