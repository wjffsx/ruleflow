package datacontext

import (
	"sync"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  MapDataContext — 应用层友好的 DataContext 实现
// ─────────────────────────────────────────────

// MapDataContext 基于 map[string]any 的轻量级 DataContext。
//
// 适用场景：
//   - 单元测试、集成测试
//   - 简单的应用层数据传入
//   - 从 JSON / Hash 反序列化的数据
//
// 字段语义遵循 DataContext 接口（详见 core/data_context.go）。
// 线程安全：所有读 / 写方法均通过 sync.Mutex 保护。
type MapDataContext struct {
	mu            sync.RWMutex
	deviceID      string
	pointName     string
	pointType     string
	value         float64
	quality       int
	tags          map[string]string
	targets       []string
	dropped       bool
	prevValue     float64
	prevSet       bool
	raw           any
	timestampFunc func() int64 // 可自定义时间戳函数，默认 time.Now().UnixNano()
}

// NewMapDataContext 创建基于 map 的 DataContext。
//
// params：
//   - m: 可选，key/value 映射。支持的 key：
//   - "device_id"   string
//   - "point_name"  string
//   - "point_type"  string
//   - "value"       float64
//   - "quality"     int
//   - "tags"        map[string]string
//   - "targets"     []string
//   - "raw"         any
func NewMapDataContext(m map[string]any) *MapDataContext {
	dc := &MapDataContext{
		pointType: "analog",
		tags:      make(map[string]string),
	}
	if m == nil {
		return dc
	}
	if v, ok := m["device_id"].(string); ok {
		dc.deviceID = v
	}
	if v, ok := m["point_name"].(string); ok {
		dc.pointName = v
	}
	if v, ok := m["point_type"].(string); ok {
		dc.pointType = v
	}
	if v, ok := m["value"]; ok {
		switch n := v.(type) {
		case float64:
			dc.value = n
		case float32:
			dc.value = float64(n)
		case int:
			dc.value = float64(n)
		case int64:
			dc.value = float64(n)
		}
	}
	if v, ok := m["quality"]; ok {
		switch n := v.(type) {
		case int:
			dc.quality = n
		case int64:
			dc.quality = int(n)
		case float64:
			dc.quality = int(n)
		}
	}
	if v, ok := m["tags"].(map[string]string); ok {
		dc.tags = v
	}
	if v, ok := m["targets"].([]string); ok {
		dc.targets = v
	}
	if v, ok := m["raw"]; ok {
		dc.raw = v
	}
	return dc
}

// DeviceID 返回设备 ID
func (m *MapDataContext) DeviceID() string { return m.deviceID }

// PointName 返回数据点名称
func (m *MapDataContext) PointName() string { return m.pointName }

// PointType 返回数据点类型
func (m *MapDataContext) PointType() string { return m.pointType }

// FQN 返回完全限定名
func (m *MapDataContext) FQN() string { return m.deviceID + "/" + m.pointName }

// Value 返回当前值
func (m *MapDataContext) Value() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.value
}

// SetValue 设置当前值
func (m *MapDataContext) SetValue(v float64) {
	m.mu.Lock()
	m.value = v
	m.mu.Unlock()
}

// Quality 返回质量码
func (m *MapDataContext) Quality() int { return m.quality }

// SetQuality 设置质量码
func (m *MapDataContext) SetQuality(q int) { m.quality = q }

// UpperLimit 返回上限（无配置时返回 0, false）
func (m *MapDataContext) UpperLimit() (float64, bool) { return 0, false }

// LowerLimit 返回下限（无配置时返回 0, false）
func (m *MapDataContext) LowerLimit() (float64, bool) { return 0, false }

// LimitExceeded 返回是否越限
func (m *MapDataContext) LimitExceeded() bool { return false }

// SetLimitExceeded 设置越限状态
func (m *MapDataContext) SetLimitExceeded(_ bool) {}

// GetTag 获取标签
func (m *MapDataContext) GetTag(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tags[key]
}

// SetTag 设置标签
func (m *MapDataContext) SetTag(key, value string) {
	m.mu.Lock()
	m.tags[key] = value
	m.mu.Unlock()
}

// TargetCount 返回目标数量
func (m *MapDataContext) TargetCount() int { return len(m.targets) }

// TargetAt 返回第 i 个目标
func (m *MapDataContext) TargetAt(i int) string {
	if i < 0 || i >= len(m.targets) {
		return ""
	}
	return m.targets[i]
}

// AddTarget 添加目标
func (m *MapDataContext) AddTarget(target string) {
	m.mu.Lock()
	m.targets = append(m.targets, target)
	m.mu.Unlock()
}

// Dropped 返回是否被丢弃
func (m *MapDataContext) Dropped() bool { return m.dropped }

// SetDropped 设置丢弃标志
func (m *MapDataContext) SetDropped(v bool) { m.dropped = v }

// Timestamp 返回时间戳
func (m *MapDataContext) Timestamp() int64 {
	if m.timestampFunc != nil {
		return m.timestampFunc()
	}
	return time.Now().UnixNano()
}

// SpanContext 返回 SpanContext（V2 零 otel 依赖）
func (m *MapDataContext) SpanContext() contract.SpanContext { return contract.SpanContext{} }

// SetSpanContext 设置 SpanContext
func (m *MapDataContext) SetSpanContext(_ contract.SpanContext) {}

// PreviousValue 返回前值（如果有）
func (m *MapDataContext) PreviousValue() (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.prevValue, m.prevSet
}

// SetPreviousValue 设置前值
func (m *MapDataContext) SetPreviousValue(v float64) {
	m.mu.Lock()
	m.prevValue = v
	m.prevSet = true
	m.mu.Unlock()
}

// Raw 返回原始数据
func (m *MapDataContext) Raw() any { return m.raw }

// SetTimestampFunc 设置自定义时间戳函数（用于测试 mock）
func (m *MapDataContext) SetTimestampFunc(f func() int64) {
	m.timestampFunc = f
}
