package ext

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  StatusChangeLogAction — 状态变更日志动作
// ─────────────────────────────────────────────

// EventStore 状态变更事件存储接口
type EventStore interface {
	Save(ctx context.Context, event any) error
}

// EventBus 事件总线接口
type EventBus interface {
	Publish(event any)
}

// StatusChangeLogAction 记录状态变更事件
type StatusChangeLogAction struct {
	IDValue    string
	EventStore EventStore
	EventBus   EventBus
}

var _ core.Action = (*StatusChangeLogAction)(nil)

func NewStatusChangeLogAction(id string, store EventStore, bus EventBus) *StatusChangeLogAction {
	return &StatusChangeLogAction{IDValue: id, EventStore: store, EventBus: bus}
}

func (a *StatusChangeLogAction) Execute(ctx context.Context, data core.DataContext) error {
	deviceID := data.DeviceID()
	pointName := data.PointName()

	changeLogEnabled := data.GetTag("changeLogEnabled")
	if changeLogEnabled != "true" || a.EventStore == nil {
		return nil
	}

	displayName := data.GetTag("displayName")
	oldValue := data.GetTag("old_value")
	newValue := data.GetTag("new_value")
	oldValueDesc := data.GetTag("oldValueDesc")
	newValueDesc := data.GetTag("newValueDesc")
	qualityStr := data.GetTag("quality")

	quality := 0
	if qualityStr != "" {
		if q, err := fmt.Sscanf(qualityStr, "%d", &quality); err != nil || q != 1 {
			quality = 0
		}
	}

	now := time.Now()
	eventID := fmt.Sprintf("sce-%d-%s", now.UnixNano(), deviceID)

	event := map[string]any{
		"id":              eventID,
		"device_id":       deviceID,
		"point_name":      pointName,
		"display_name":    displayName,
		"old_value":       oldValue,
		"new_value":       newValue,
		"old_value_desc":  oldValueDesc,
		"new_value_desc":  newValueDesc,
		"quality":         quality,
		"occurred_at":     now,
		"changed_enabled": true,
	}

	// 同步写存储（异步由调用方控制）
	_ = a.EventStore.Save(context.Background(), event)

	// 事件总线发布
	if a.EventBus != nil {
		a.EventBus.Publish(event)
	}

	return nil
}

func (a *StatusChangeLogAction) ID() string          { return a.IDValue }
func (a *StatusChangeLogAction) Type() string        { return "status_change_log" }
func (a *StatusChangeLogAction) Description() string { return "status change log" }
