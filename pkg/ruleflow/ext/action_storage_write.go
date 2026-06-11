package ext

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  共享存储接口
// ─────────────────────────────────────────────

// Storage 数据写入接口
type Storage interface {
	WriteData(dp any) error
	WriteDataBatch(dps []any) error
}

// OutboxPublisher 事务发件箱接口
type OutboxPublisher interface {
	CreateMessage(sourceID string, eventType string, payload any) (any, error)
	Save(ctx context.Context, msg any) error
}

// ─────────────────────────────────────────────
//  StorageWriteAction — 存储写入动作
// ─────────────────────────────────────────────

// StorageWriteAction 将处理后的数据写入实时库
type StorageWriteAction struct {
	IDValue         string
	Target          string
	Storage         Storage
	OutboxPublisher OutboxPublisher
}

var _ core.Action = (*StorageWriteAction)(nil)

func NewStorageWriteAction(id, target string, storage Storage, pub OutboxPublisher) *StorageWriteAction {
	return &StorageWriteAction{
		IDValue:         id,
		Target:          target,
		Storage:         storage,
		OutboxPublisher: pub,
	}
}

func (a *StorageWriteAction) Execute(_ context.Context, data core.DataContext) error {
	deviceID := data.DeviceID()
	pointName := data.PointName()
	if deviceID == "" || pointName == "" {
		return nil
	}

	value := data.Value()
	ts := data.Timestamp()
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	quality := data.Quality()

	dp := map[string]any{
		"device_id":  deviceID,
		"point_name": pointName,
		"value":      value,
		"timestamp":  ts,
		"quality":    quality,
	}

	// 计算结果的覆盖
	if cr := data.GetTag("calc_result"); cr != "" {
		if f, err := strconv.ParseFloat(cr, 64); err == nil {
			dp["value"] = f
		}
	}

	// 如果指定了 target，做路由
	if a.Target != "" {
		data.AddTarget(a.Target)
	}

	if a.Storage == nil {
		return nil
	}

	if err := a.Storage.WriteData(dp); err != nil {
		return fmt.Errorf("storage write: %w", err)
	}

	// outbox 事务消息
	if a.OutboxPublisher != nil {
		notif := map[string]any{
			"device_id":  deviceID,
			"point_name": pointName,
			"value":      value,
			"timestamp":  time.UnixMilli(ts),
			"quality":    strconv.Itoa(quality),
			"has_value":  true,
		}
		outboxMsg, err := a.OutboxPublisher.CreateMessage(deviceID, "data_updated", notif)
		if err == nil {
			_ = a.OutboxPublisher.Save(context.Background(), outboxMsg)
		}
	}

	return nil
}

func (a *StorageWriteAction) ID() string          { return a.IDValue }
func (a *StorageWriteAction) Type() string        { return "storage_write" }
func (a *StorageWriteAction) Description() string { return "storage write" }
