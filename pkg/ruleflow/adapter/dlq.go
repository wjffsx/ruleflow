package adapter

import (
	"context"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DLQAdapter — 数据丢失追踪器适配器
// ─────────────────────────────────────────────

// DLQ 是 DLQ 的窄接口视图
type DLQ interface {
	RecordDrop(deviceID, pointName string, timestamp int64, reason string)
	RecordError(deviceID, pointName string, timestamp int64, errMsg string)
}

// DLQAdapter 将 DLQ 适配为 ruleflow 的 DataLossTracker
type DLQAdapter struct {
	dlq DLQ
}

// NewDLQAdapter 创建 DLQ 适配器
func NewDLQAdapter(dlq DLQ) *DLQAdapter {
	return &DLQAdapter{dlq: dlq}
}

// TrackDrop 追踪数据丢弃
func (a *DLQAdapter) TrackDrop(ctx context.Context, data any, ruleID string, reason string) {
	if dc, ok := data.(core.DataContext); ok {
		a.dlq.RecordDrop(dc.DeviceID(), dc.PointName(), dc.Timestamp(), reason)
		return
	}
	// 非 DataContext 类型时，记录 best-effort 信息
	if dc, ok := data.(interface {
		DeviceID() string
		PointName() string
		Timestamp() (int64, error)
	}); ok {
		ts, _ := dc.Timestamp()
		a.dlq.RecordDrop(dc.DeviceID(), dc.PointName(), ts, reason)
	}
}

// TrackError 追踪规则执行错误
func (a *DLQAdapter) TrackError(ctx context.Context, data any, ruleID string, err error) {
	if dc, ok := data.(core.DataContext); ok {
		a.dlq.RecordError(dc.DeviceID(), dc.PointName(), dc.Timestamp(), err.Error())
		return
	}
}
