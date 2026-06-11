package router

import (
	"context"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  测试辅助函数
// ─────────────────────────────────────────────

func mockDataContext(pointName, pointType string, value float64) core.DataContext {
	return &mockData{
		pointName: pointName,
		pointType: pointType,
		value:     value,
		deviceID:  "device_001",
		fqn:       "device_001/" + pointName,
		quality:   192,
		timestamp: 1234567890,
		tags:      make(map[string]string),
		targets:   []string{},
	}
}

type mockData struct {
	deviceID      string
	pointName     string
	pointType     string
	fqn           string
	value         float64
	quality       int
	timestamp     int64
	tags          map[string]string
	targets       []string
	dropped       bool
	limitExceeded bool
	upperLimit    float64
	hasUpperLimit bool
	lowerLimit    float64
	hasLowerLimit bool
	prevValue     float64
	hasPrevValue  bool
	spanContext   contract.SpanContext
}

func (m *mockData) DeviceID() string                        { return m.deviceID }
func (m *mockData) PointName() string                       { return m.pointName }
func (m *mockData) PointType() string                       { return m.pointType }
func (m *mockData) FQN() string                             { return m.fqn }
func (m *mockData) Value() float64                          { return m.value }
func (m *mockData) SetValue(v float64)                      { m.value = v }
func (m *mockData) Quality() int                            { return m.quality }
func (m *mockData) SetQuality(q int)                        { m.quality = q }
func (m *mockData) Timestamp() int64                        { return m.timestamp }
func (m *mockData) Dropped() bool                           { return m.dropped }
func (m *mockData) SetDropped(v bool)                       { m.dropped = v }
func (m *mockData) LimitExceeded() bool                     { return m.limitExceeded }
func (m *mockData) SetLimitExceeded(v bool)                 { m.limitExceeded = v }
func (m *mockData) UpperLimit() (float64, bool)             { return m.upperLimit, m.hasUpperLimit }
func (m *mockData) LowerLimit() (float64, bool)             { return m.lowerLimit, m.hasLowerLimit }
func (m *mockData) GetTag(key string) string                { return m.tags[key] }
func (m *mockData) SetTag(key, value string)                { m.tags[key] = value }
func (m *mockData) TargetCount() int                        { return len(m.targets) }
func (m *mockData) TargetAt(i int) string                   { return m.targets[i] }
func (m *mockData) AddTarget(target string)                 { m.targets = append(m.targets, target) }
func (m *mockData) PreviousValue() (float64, bool)          { return m.prevValue, m.hasPrevValue }
func (m *mockData) SetPreviousValue(v float64)              { m.prevValue = v; m.hasPrevValue = true }
func (m *mockData) SpanContext() contract.SpanContext      { return m.spanContext }
func (m *mockData) SetSpanContext(sc contract.SpanContext) { m.spanContext = sc }
func (m *mockData) Raw() any                                { return nil }

// ─────────────────────────────────────────────
//  DataRouter 测试
// ─────────────────────────────────────────────

func TestDataRouter_RegisterChain(t *testing.T) {
	router := NewDataRouter()

	chain := &core.RuleChain{
		ID:           "chain_analog_001",
		Name:         "温度监控链",
		PipelineType: "analog",
		Inputs: []core.RuleChainInput{
			{PointName: "temp_001", DisplayName: "温度1", PointType: "analog", DataType: "double"},
			{PointName: "temp_002", DisplayName: "温度2", PointType: "analog", DataType: "double"},
		},
	}

	err := router.RegisterChain(chain)
	if err != nil {
		t.Fatalf("RegisterChain failed: %v", err)
	}

	// 验证路由条目
	entry, ok := router.GetRouterEntry(chain.ID)
	if !ok {
		t.Fatal("RouterEntry not found")
	}

	if entry.ChainID != chain.ID {
		t.Errorf("ChainID mismatch: got %s, want %s", entry.ChainID, chain.ID)
	}

	if entry.PipelineType != "analog" {
		t.Errorf("PipelineType mismatch: got %s, want analog", entry.PipelineType)
	}

	if entry.InputCount != 2 {
		t.Errorf("InputCount mismatch: got %d, want 2", entry.InputCount)
	}

	// 验证输入索引
	if idx, ok := entry.InputIndex["temp_001"]; !ok || idx != 0 {
		t.Errorf("InputIndex[temp_001] mismatch: got %d, want 0", idx)
	}

	if idx, ok := entry.InputIndex["temp_002"]; !ok || idx != 1 {
		t.Errorf("InputIndex[temp_002] mismatch: got %d, want 1", idx)
	}
}

func TestDataRouter_Route(t *testing.T) {
	router := NewDataRouter()

	// 注册模拟量链
	chainAnalog := &core.RuleChain{
		ID:           "chain_analog_001",
		PipelineType: "analog",
		Inputs: []core.RuleChainInput{
			{PointName: "temp_001", PointType: "analog"},
			{PointName: "temp_002", PointType: "analog"},
		},
	}
	router.RegisterChain(chainAnalog)

	// 注册数字量链
	chainDigital := &core.RuleChain{
		ID:           "chain_digital_001",
		PipelineType: "digital",
		Inputs: []core.RuleChainInput{
			{PointName: "switch_001", PointType: "digital"},
		},
	}
	router.RegisterChain(chainDigital)

	// 注册无类型链
	chainUntyped := &core.RuleChain{
		ID:           "chain_untyped_001",
		PipelineType: "",
		Inputs: []core.RuleChainInput{
			{PointName: "any_001", PointType: "analog"},
		},
	}
	router.RegisterChain(chainUntyped)

	ctx := context.Background()

	// 测试 1: 模拟量数据路由到模拟量链
	results, err := router.Route(ctx, mockDataContext("temp_001", "analog", 25.5))
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Route results count mismatch: got %d, want 1", len(results))
	}

	if results[0].ChainID != "chain_analog_001" {
		t.Errorf("Route ChainID mismatch: got %s, want chain_analog_001", results[0].ChainID)
	}

	if !results[0].MatchedType {
		t.Error("Route MatchedType should be true")
	}

	if !results[0].Declared {
		t.Error("Route Declared should be true")
	}

	// 测试 2: 数字量数据路由到数字量链
	results, err = router.Route(ctx, mockDataContext("switch_001", "digital", 1.0))
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Route results count mismatch: got %d, want 1", len(results))
	}

	if results[0].ChainID != "chain_digital_001" {
		t.Errorf("Route ChainID mismatch: got %s, want chain_digital_001", results[0].ChainID)
	}

	// 测试 3: 类型不匹配的数据不应路由
	results, err = router.Route(ctx, mockDataContext("temp_001", "digital", 1.0))
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// 数字量 temp_001 不应路由到模拟量链
	if len(results) != 0 {
		t.Errorf("Type mismatch should not route: got %d results, want 0", len(results))
	}

	// 测试 4: 未声明的数据点不应路由
	results, err = router.Route(ctx, mockDataContext("unknown_001", "analog", 10.0))
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Undeclared point should not route: got %d results, want 0", len(results))
	}

	// 测试 5: 无类型链应接受任意类型数据
	results, err = router.Route(ctx, mockDataContext("any_001", "analog", 10.0))
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Untyped chain should accept data: got %d results, want 1", len(results))
	}

	if results[0].ChainID != "chain_untyped_001" {
		t.Errorf("Untyped chain route mismatch: got %s, want chain_untyped_001", results[0].ChainID)
	}
}

func TestDataRouter_RouteToChain(t *testing.T) {
	router := NewDataRouter()

	chain := &core.RuleChain{
		ID:           "chain_analog_001",
		PipelineType: "analog",
		Inputs: []core.RuleChainInput{
			{PointName: "temp_001", PointType: "analog"},
		},
	}
	router.RegisterChain(chain)

	ctx := context.Background()

	// 测试 1: 正确路由
	result, err := router.RouteToChain(ctx, "chain_analog_001", mockDataContext("temp_001", "analog", 25.5))
	if err != nil {
		t.Fatalf("RouteToChain failed: %v", err)
	}

	if !result.MatchedType {
		t.Error("RouteToChain MatchedType should be true")
	}

	if !result.Declared {
		t.Error("RouteToChain Declared should be true")
	}

	// 测试 2: 类型不匹配
	result, err = router.RouteToChain(ctx, "chain_analog_001", mockDataContext("temp_001", "digital", 1.0))
	if err != nil {
		t.Fatalf("RouteToChain failed: %v", err)
	}

	if result.MatchedType {
		t.Error("Type mismatch should have MatchedType=false")
	}

	// 测试 3: 未声明输入
	result, err = router.RouteToChain(ctx, "chain_analog_001", mockDataContext("unknown_001", "analog", 10.0))
	if err != nil {
		t.Fatalf("RouteToChain failed: %v", err)
	}

	if result.Declared {
		t.Error("Undeclared input should have Declared=false")
	}

	// 测试 4: 未注册链
	_, err = router.RouteToChain(ctx, "unknown_chain", mockDataContext("temp_001", "analog", 25.5))
	if err == nil {
		t.Error("Unknown chain should return error")
	}
}

func TestDataRouter_ValidateRoute(t *testing.T) {
	router := NewDataRouter()

	chain := &core.RuleChain{
		ID:           "chain_analog_001",
		PipelineType: "analog",
		Inputs: []core.RuleChainInput{
			{PointName: "temp_001", PointType: "analog"},
		},
	}
	router.RegisterChain(chain)

	ctx := context.Background()

	// 测试 1: 有效路由
	err := router.ValidateRoute(ctx, "chain_analog_001", mockDataContext("temp_001", "analog", 25.5))
	if err != nil {
		t.Errorf("Valid route should not return error: %v", err)
	}

	// 测试 2: 类型不匹配
	err = router.ValidateRoute(ctx, "chain_analog_001", mockDataContext("temp_001", "digital", 1.0))
	if err == nil {
		t.Error("Type mismatch should return error")
	}

	routeErr, ok := err.(*RouteError)
	if !ok {
		t.Error("Error should be RouteError type")
	}

	if routeErr.Declared != true {
		t.Error("RouteError Declared should be true for type mismatch")
	}

	// 测试 3: 未声明输入
	err = router.ValidateRoute(ctx, "chain_analog_001", mockDataContext("unknown_001", "analog", 10.0))
	if err == nil {
		t.Error("Undeclared input should return error")
	}

	routeErr, ok = err.(*RouteError)
	if !ok {
		t.Error("Error should be RouteError type")
	}

	if routeErr.Declared != false {
		t.Error("RouteError Declared should be false for undeclared input")
	}
}

func TestDataRouter_UnregisterChain(t *testing.T) {
	router := NewDataRouter()

	chain := &core.RuleChain{
		ID:           "chain_001",
		PipelineType: "analog",
		Inputs: []core.RuleChainInput{
			{PointName: "temp_001", PointType: "analog"},
		},
	}
	router.RegisterChain(chain)

	// 验证注册成功
	_, ok := router.GetRouterEntry(chain.ID)
	if !ok {
		t.Fatal("Chain should be registered")
	}

	// 取消注册
	router.UnregisterChain(chain.ID)

	// 验证取消注册成功
	_, ok = router.GetRouterEntry(chain.ID)
	if ok {
		t.Error("Chain should be unregistered")
	}

	// 路由不应返回结果
	ctx := context.Background()
	results, err := router.Route(ctx, mockDataContext("temp_001", "analog", 25.5))
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Unregistered chain should not route: got %d results, want 0", len(results))
	}
}
