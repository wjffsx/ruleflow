// Package adapter 提供 DataContext 适配器实现。
//
// V12 架构边界说明：
//   - 本包提供 DataContextAdapter，用于将外部 DataPoint 适配为 DataContext 接口
//   - 零拷贝设计：直接委托到原始数据点，不复制任何字段
//   - 生产环境推荐使用此适配器，避免数据复制开销
//
// 职责边界：
//   - adapter: 外部数据点适配（零拷贝，依赖 DataPoint 接口）
//   - datacontext: 内部实现（MapDataContext, MultiDataContext），用于测试和示例
//
// 使用场景：
//   - IoT 数据点接入：将 UnifiedDataPoint 适配为 DataContext
//   - 高性能场景：零拷贝设计减少内存分配
package adapter

import (
	"sync"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  DataContextAdapter — UnifiedDataPoint 零拷贝适配器
// ─────────────────────────────────────────────

// DataPoint 是 UnifiedDataPoint 的最小接口视图。
// ruleflow 仅依赖此窄接口，不直接依赖上层 models 包。
type DataPoint interface {
	GetDeviceID() string
	GetPointName() string
	SetPointName(name string) // 新增：支持 RenameAction
	GetPointType() string
	GetValue() float64
	SetValue(v float64)
	GetQuality() int
	SetQuality(q int)
	GetUpperLimit() (float64, bool)
	GetLowerLimit() (float64, bool)
	IsLimitExceeded() bool
	SetLimitExceeded(v bool)
	GetTimestamp() int64
	IsDropped() bool
	SetDropped(v bool)
	GetGroup() string
}

// DataContextAdapter 将 DataPoint 适配为 ruleflow 的 DataContext 接口。
// 零拷贝设计：直接委托到原始数据点，不复制任何字段。
type DataContextAdapter struct {
	point    DataPoint
	targets  []string          // 预分配，避免热路径分配
	tags     map[string]string // 延迟分配
	fqn      string            // 预计算缓存：DeviceID/PointName
	prevVal  float64           // 前值缓存（值类型，避免堆分配）
	prevSet  bool              // 前值是否已设置
}

// 对象池
var dataCtxPool = sync.Pool{
	New: func() any {
		return &DataContextAdapter{
			targets: make([]string, 0, 4),
		}
	},
}

// AcquireDataContext 从对象池获取适配器
func AcquireDataContext(point DataPoint) *DataContextAdapter {
	ctx := dataCtxPool.Get().(*DataContextAdapter)
	ctx.point = point
	ctx.targets = ctx.targets[:0]
	ctx.fqn = point.GetDeviceID() + "/" + point.GetPointName()
	return ctx
}

// ReleaseDataContext 归还到对象池
func ReleaseDataContext(ctx *DataContextAdapter) {
	ctx.point = nil
	ctx.tags = nil
	ctx.fqn = ""
	ctx.prevVal = 0
	ctx.prevSet = false
	dataCtxPool.Put(ctx)
}

// 实现 DataContext 接口

func (c *DataContextAdapter) DeviceID() string            { return c.point.GetDeviceID() }
func (c *DataContextAdapter) PointName() string           { return c.point.GetPointName() }
func (c *DataContextAdapter) SetPointName(name string) {
	c.point.SetPointName(name)
	c.fqn = c.point.GetDeviceID() + "/" + name // 更新缓存
}
func (c *DataContextAdapter) PointType() string           { return c.point.GetPointType() }
func (c *DataContextAdapter) FQN() string                 { return c.fqn }
func (c *DataContextAdapter) Value() float64              { return c.point.GetValue() }
func (c *DataContextAdapter) SetValue(v float64)          { c.point.SetValue(v) }
func (c *DataContextAdapter) Quality() int                { return c.point.GetQuality() }
func (c *DataContextAdapter) SetQuality(q int)            { c.point.SetQuality(q) }
func (c *DataContextAdapter) UpperLimit() (float64, bool) { return c.point.GetUpperLimit() }
func (c *DataContextAdapter) LowerLimit() (float64, bool) { return c.point.GetLowerLimit() }
func (c *DataContextAdapter) LimitExceeded() bool         { return c.point.IsLimitExceeded() }
func (c *DataContextAdapter) SetLimitExceeded(v bool)     { c.point.SetLimitExceeded(v) }
func (c *DataContextAdapter) Timestamp() int64            { return c.point.GetTimestamp() }
func (c *DataContextAdapter) Dropped() bool               { return c.point.IsDropped() }
func (c *DataContextAdapter) SetDropped(v bool)           { c.point.SetDropped(v) }

func (c *DataContextAdapter) GetTag(key string) string {
	if c.tags == nil {
		return ""
	}
	return c.tags[key]
}

func (c *DataContextAdapter) SetTag(key, value string) {
	if c.tags == nil {
		c.tags = make(map[string]string, 4)
	}
	c.tags[key] = value
}

func (c *DataContextAdapter) TargetCount() int { return len(c.targets) }
func (c *DataContextAdapter) TargetAt(i int) string {
	if i < 0 || i >= len(c.targets) {
		return ""
	}
	return c.targets[i]
}
func (c *DataContextAdapter) AddTarget(target string) { c.targets = append(c.targets, target) }

func (c *DataContextAdapter) SpanContext() contract.SpanContext     { return contract.SpanContext{} }
func (c *DataContextAdapter) SetSpanContext(_ contract.SpanContext) {}
func (c *DataContextAdapter) Raw() any                              { return c.point }

func (c *DataContextAdapter) PreviousValue() (float64, bool) {
	return c.prevVal, c.prevSet
}

func (c *DataContextAdapter) SetPreviousValue(v float64) {
	c.prevVal = v
	c.prevSet = true
}
