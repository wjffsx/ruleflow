package datacontext

import (
	"context"
	"testing"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  测试辅助函数
// ─────────────────────────────────────────────

func mockDataContextForMulti(pointName string, value float64) core.DataContext {
	return &mockDataForMulti{
		pointName: pointName,
		value:     value,
		deviceID:  "device_001",
	}
}

type mockDataForMulti struct {
	pointName string
	value     float64
	deviceID  string
}

func (m *mockDataForMulti) DeviceID() string                       { return m.deviceID }
func (m *mockDataForMulti) PointName() string                      { return m.pointName }
func (m *mockDataForMulti) PointType() string                      { return "analog" }
func (m *mockDataForMulti) FQN() string                            { return m.deviceID + "/" + m.pointName }
func (m *mockDataForMulti) Value() float64                         { return m.value }
func (m *mockDataForMulti) SetValue(v float64)                     { m.value = v }
func (m *mockDataForMulti) Quality() int                           { return 192 }
func (m *mockDataForMulti) SetQuality(q int)                       {}
func (m *mockDataForMulti) Timestamp() int64                       { return time.Now().UnixNano() }
func (m *mockDataForMulti) Dropped() bool                          { return false }
func (m *mockDataForMulti) SetDropped(v bool)                      {}
func (m *mockDataForMulti) LimitExceeded() bool                    { return false }
func (m *mockDataForMulti) SetLimitExceeded(v bool)                {}
func (m *mockDataForMulti) UpperLimit() (float64, bool)            { return 0, false }
func (m *mockDataForMulti) LowerLimit() (float64, bool)            { return 0, false }
func (m *mockDataForMulti) GetTag(key string) string               { return "" }
func (m *mockDataForMulti) SetTag(key, value string)               {}
func (m *mockDataForMulti) TargetCount() int                       { return 0 }
func (m *mockDataForMulti) TargetAt(i int) string                  { return "" }
func (m *mockDataForMulti) AddTarget(target string)                {}
func (m *mockDataForMulti) PreviousValue() (float64, bool)         { return 0, false }
func (m *mockDataForMulti) SetPreviousValue(v float64)             {}
func (m *mockDataForMulti) SpanContext() contract.SpanContext      { return contract.SpanContext{} }
func (m *mockDataForMulti) SetSpanContext(sc contract.SpanContext) {}
func (m *mockDataForMulti) Raw() any                               { return nil }

// ─────────────────────────────────────────────
//  MultiDataContext 测试
// ─────────────────────────────────────────────

func TestMultiDataContext_New(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002", "temp_003"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)

	if mdc.DeviceID != "device_001" {
		t.Errorf("DeviceID mismatch: got %s, want device_001", mdc.DeviceID)
	}

	if mdc.ChainID != "chain_001" {
		t.Errorf("ChainID mismatch: got %s, want chain_001", mdc.ChainID)
	}

	if mdc.Total != 3 {
		t.Errorf("Total mismatch: got %d, want 3", mdc.Total)
	}

	if mdc.Count != 0 {
		t.Errorf("Count should be 0, got %d", mdc.Count)
	}

	if mdc.IsReady() {
		t.Error("IsReady should be false")
	}
}

func TestMultiDataContext_AddPoint(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)

	// 添加第一个点
	ready := mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 25.5))
	if ready {
		t.Error("Should not be ready after first point")
	}

	if mdc.Count != 1 {
		t.Errorf("Count should be 1, got %d", mdc.Count)
	}

	// 添加第二个点
	ready = mdc.AddPoint("temp_002", mockDataContextForMulti("temp_002", 26.0))
	if !ready {
		t.Error("Should be ready after second point")
	}

	if mdc.Count != 2 {
		t.Errorf("Count should be 2, got %d", mdc.Count)
	}

	if !mdc.IsReady() {
		t.Error("IsReady should be true")
	}
}

func TestMultiDataContext_AddPoint_Undeclared(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)

	// 添加未声明的点
	ready := mdc.AddPoint("unknown_001", mockDataContextForMulti("unknown_001", 10.0))
	if ready {
		t.Error("Undeclared point should not trigger ready")
	}

	if mdc.Count != 0 {
		t.Errorf("Count should remain 0, got %d", mdc.Count)
	}
}

func TestMultiDataContext_GetPoint(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)
	mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 25.5))

	// 获取已添加的点
	data, ok := mdc.GetPoint("temp_001")
	if !ok {
		t.Error("temp_001 should exist")
	}

	if data.Value() != 25.5 {
		t.Errorf("Value mismatch: got %f, want 25.5", data.Value())
	}

	// 获取未添加的点
	data, ok = mdc.GetPoint("temp_002")
	if ok {
		t.Error("temp_002 should not exist")
	}
}

func TestMultiDataContext_GetMissingInputs(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002", "temp_003"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)
	mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 25.5))

	missing := mdc.GetMissingInputs()
	if len(missing) != 2 {
		t.Errorf("Missing inputs count mismatch: got %d, want 2", len(missing))
	}

	// 验证缺失的输入点
	found := make(map[string]bool)
	for _, name := range missing {
		found[name] = true
	}

	if !found["temp_002"] || !found["temp_003"] {
		t.Error("Missing inputs should contain temp_002 and temp_003")
	}
}

func TestMultiDataContext_Reset(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)
	mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 25.5))
	mdc.AddPoint("temp_002", mockDataContextForMulti("temp_002", 26.0))

	// 重置
	mdc.Reset()

	if mdc.Count != 0 {
		t.Errorf("Count should be 0 after reset, got %d", mdc.Count)
	}

	if mdc.IsReady() {
		t.Error("IsReady should be false after reset")
	}
}

func TestMultiDataContext_AverageValue(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002", "temp_003"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)
	mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 20.0))
	mdc.AddPoint("temp_002", mockDataContextForMulti("temp_002", 30.0))
	mdc.AddPoint("temp_003", mockDataContextForMulti("temp_003", 40.0))

	avg := mdc.AverageValue()
	expected := 30.0

	if avg != expected {
		t.Errorf("AverageValue mismatch: got %f, want %f", avg, expected)
	}
}

func TestMultiDataContext_MaxValue(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002", "temp_003"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)
	mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 20.0))
	mdc.AddPoint("temp_002", mockDataContextForMulti("temp_002", 50.0))
	mdc.AddPoint("temp_003", mockDataContextForMulti("temp_003", 30.0))

	max := mdc.MaxValue()
	expected := 50.0

	if max != expected {
		t.Errorf("MaxValue mismatch: got %f, want %f", max, expected)
	}
}

func TestMultiDataContext_MinValue(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002", "temp_003"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)
	mdc.AddPoint("temp_001", mockDataContextForMulti("temp_001", 20.0))
	mdc.AddPoint("temp_002", mockDataContextForMulti("temp_002", 50.0))
	mdc.AddPoint("temp_003", mockDataContextForMulti("temp_003", 10.0))

	min := mdc.MinValue()
	expected := 10.0

	if min != expected {
		t.Errorf("MinValue mismatch: got %f, want %f", min, expected)
	}
}

func TestMultiDataContext_AllTrue(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"switch_001", "switch_002"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)

	// 测试全为 true
	mdc.AddPoint("switch_001", mockDataContextForMulti("switch_001", 1.0))
	mdc.AddPoint("switch_002", mockDataContextForMulti("switch_002", 1.0))

	if !mdc.AllTrue() {
		t.Error("AllTrue should be true when all values are 1")
	}

	// 测试不全为 true
	mdc.Reset()
	mdc.AddPoint("switch_001", mockDataContextForMulti("switch_001", 1.0))
	mdc.AddPoint("switch_002", mockDataContextForMulti("switch_002", 0.0))

	if mdc.AllTrue() {
		t.Error("AllTrue should be false when some values are 0")
	}
}

func TestMultiDataContext_AnyTrue(t *testing.T) {
	ctx := context.Background()
	inputNames := []string{"switch_001", "switch_002"}

	mdc := NewMultiDataContext(ctx, "device_001", "chain_001", inputNames)

	// 测试有 true
	mdc.AddPoint("switch_001", mockDataContextForMulti("switch_001", 1.0))
	mdc.AddPoint("switch_002", mockDataContextForMulti("switch_002", 0.0))

	if !mdc.AnyTrue() {
		t.Error("AnyTrue should be true when some values are 1")
	}

	// 测试全为 false
	mdc.Reset()
	mdc.AddPoint("switch_001", mockDataContextForMulti("switch_001", 0.0))
	mdc.AddPoint("switch_002", mockDataContextForMulti("switch_002", 0.0))

	if mdc.AnyTrue() {
		t.Error("AnyTrue should be false when all values are 0")
	}
}

// ─────────────────────────────────────────────
//  MultiDataContextPool 测试
// ─────────────────────────────────────────────

func TestMultiDataContextPool_AcquireRelease(t *testing.T) {
	pool := NewMultiDataContextPool()
	ctx := context.Background()
	inputNames := []string{"temp_001", "temp_002"}

	// Acquire
	mdc := pool.Acquire(ctx, "device_001", "chain_001", inputNames)
	if mdc == nil {
		t.Fatal("Acquire should return non-nil MultiDataContext")
	}

	if mdc.DeviceID != "device_001" {
		t.Errorf("DeviceID mismatch: got %s, want device_001", mdc.DeviceID)
	}

	// Release
	pool.Release(mdc)

	// 再次 Acquire（应该从池中获取）
	mdc2 := pool.Acquire(ctx, "device_002", "chain_002", inputNames)
	if mdc2 == nil {
		t.Fatal("Second Acquire should return non-nil MultiDataContext")
	}

	if mdc2.DeviceID != "device_002" {
		t.Errorf("DeviceID should be reset: got %s, want device_002", mdc2.DeviceID)
	}

	pool.Release(mdc2)
}

// ─────────────────────────────────────────────
//  MultiInputBuffer 测试
// ─────────────────────────────────────────────

func TestMultiInputBuffer_New(t *testing.T) {
	ctx := context.Background()
	buf := NewMultiInputBuffer(ctx, 5*time.Second)

	if buf.timeout != 5*time.Second {
		t.Errorf("Timeout mismatch: got %v, want 5s", buf.timeout)
	}

	if buf.pool == nil {
		t.Error("Pool should not be nil")
	}

	buf.Stop()
}

func TestMultiInputBuffer_Add(t *testing.T) {
	ctx := context.Background()
	buf := NewMultiInputBuffer(ctx, 5*time.Second)

	triggered := false
	buf.SetTriggerCallback(func(ctx context.Context, mdc *MultiDataContext) {
		triggered = true
	})

	inputNames := []string{"temp_001", "temp_002"}

	// 添加第一个点
	ready := buf.Add("chain_001", inputNames, mockDataContextForMulti("temp_001", 25.5))
	if ready {
		t.Error("Should not be ready after first point")
	}

	// 添加第二个点
	ready = buf.Add("chain_001", inputNames, mockDataContextForMulti("temp_002", 26.0))
	if !ready {
		t.Error("Should be ready after second point")
	}

	// 等待回调触发
	time.Sleep(100 * time.Millisecond)

	if !triggered {
		t.Error("Trigger callback should be called")
	}

	buf.Stop()
}

func TestMultiInputBuffer_Add_Undeclared(t *testing.T) {
	ctx := context.Background()
	buf := NewMultiInputBuffer(ctx, 5*time.Second)

	inputNames := []string{"temp_001", "temp_002"}

	// 添加未声明的点
	ready := buf.Add("chain_001", inputNames, mockDataContextForMulti("unknown_001", 10.0))
	if ready {
		t.Error("Undeclared point should not trigger ready")
	}

	buf.Stop()
}

func TestMultiInputBuffer_Size(t *testing.T) {
	ctx := context.Background()
	buf := NewMultiInputBuffer(ctx, 5*time.Second)

	// 设置触发回调（确保清理逻辑执行）
	triggered := false
	buf.SetTriggerCallback(func(ctx context.Context, mdc *MultiDataContext) {
		triggered = true
	})

	inputNames := []string{"temp_001", "temp_002"}

	// 添加第一个点
	buf.Add("chain_001", inputNames, mockDataContextForMulti("temp_001", 25.5))

	if buf.Size() != 1 {
		t.Errorf("Size should be 1, got %d", buf.Size())
	}

	// 添加第二个点（触发清理）
	buf.Add("chain_001", inputNames, mockDataContextForMulti("temp_002", 26.0))

	// 等待清理（异步）
	time.Sleep(200 * time.Millisecond)

	if !triggered {
		t.Error("Trigger callback should be called")
	}

	if buf.Size() != 0 {
		t.Errorf("Size should be 0 after trigger, got %d", buf.Size())
	}

	buf.Stop()
}

func TestMultiInputBuffer_Clear(t *testing.T) {
	ctx := context.Background()
	buf := NewMultiInputBuffer(ctx, 5*time.Second)

	inputNames := []string{"temp_001", "temp_002"}

	// 添加数据点
	buf.Add("chain_001", inputNames, mockDataContextForMulti("temp_001", 25.5))

	if buf.Size() != 1 {
		t.Errorf("Size should be 1, got %d", buf.Size())
	}

	// 清空
	buf.Clear()

	if buf.Size() != 0 {
		t.Errorf("Size should be 0 after clear, got %d", buf.Size())
	}

	buf.Stop()
}
