// Package action provides builtin action nodes
package action

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  DelayAction — 延时执行动作
// ─────────────────────────────────────────────

// DelayAction 延时执行内嵌动作。
// 注意：延时动作为异步执行，EvalChain 不会等待延时动作完成。
type DelayAction struct {
	IDValue  string
	Duration time.Duration
	Inner    core.Action // 内嵌动作
}

func NewDelayAction(id string, duration time.Duration, inner core.Action) *DelayAction {
	return &DelayAction{IDValue: id, Duration: duration, Inner: inner}
}

func (a *DelayAction) Execute(ctx context.Context, data core.DataContext) error {
	if a.Inner == nil {
		return nil
	}

	// 快照关键数据（DataContext 可能在延时后被回收）
	snapshot := &dataSnapshot{
		deviceID:  data.DeviceID(),
		pointName: data.PointName(),
		value:     data.Value(),
		quality:   data.Quality(),
		timestamp: data.Timestamp(),
	}
	inner := a.Inner

	time.AfterFunc(a.Duration, func() {
		// 使用快照数据创建只读 DataContext 执行内嵌动作
		_ = inner.Execute(context.Background(), snapshot)
	})

	return nil // 立即返回，不等待延时
}

func (a *DelayAction) ID() string          { return a.IDValue }
func (a *DelayAction) Type() string        { return "delay" }
func (a *DelayAction) Description() string { return fmt.Sprintf("delay %s", a.Duration) }

// dataSnapshot 只读数据快照，用于延时动作
type dataSnapshot struct {
	deviceID  string
	pointName string
	value     float64
	quality   int
	timestamp int64
}

func (d *dataSnapshot) DeviceID() string                    { return d.deviceID }
func (d *dataSnapshot) PointName() string                   { return d.pointName }
func (d *dataSnapshot) SetPointName(_ string)               {} // 只读快照，不支持修改
func (d *dataSnapshot) PointType() string                   { return "" }
func (d *dataSnapshot) FQN() string                         { return d.deviceID + "/" + d.pointName }
func (d *dataSnapshot) Value() float64                      { return d.value }
func (d *dataSnapshot) SetValue(v float64)                  { d.value = v }
func (d *dataSnapshot) Quality() int                        { return d.quality }
func (d *dataSnapshot) SetQuality(q int)                    { d.quality = q }
func (d *dataSnapshot) UpperLimit() (float64, bool)         { return 0, false }
func (d *dataSnapshot) LowerLimit() (float64, bool)         { return 0, false }
func (d *dataSnapshot) LimitExceeded() bool                 { return false }
func (d *dataSnapshot) SetLimitExceeded(bool)               {}
func (d *dataSnapshot) GetTag(string) string                { return "" }
func (d *dataSnapshot) SetTag(string, string)               {}
func (d *dataSnapshot) TargetCount() int                    { return 0 }
func (d *dataSnapshot) TargetAt(int) string                 { return "" }
func (d *dataSnapshot) AddTarget(string)                    {}
func (d *dataSnapshot) Dropped() bool                       { return false }
func (d *dataSnapshot) SetDropped(bool)                     {}
func (d *dataSnapshot) Timestamp() int64                    { return d.timestamp }
func (d *dataSnapshot) PreviousValue() (float64, bool)      { return 0, false }
func (d *dataSnapshot) SetPreviousValue(float64)            {}
func (d *dataSnapshot) SpanContext() contract.SpanContext   { return contract.SpanContext{} }
func (d *dataSnapshot) SetSpanContext(contract.SpanContext) {}
func (d *dataSnapshot) Raw() any                            { return nil }

// 确保 dataSnapshot 实现 DataContext 接口
var _ core.DataContext = (*dataSnapshot)(nil)
