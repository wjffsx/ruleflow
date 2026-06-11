package ext

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  AlarmNotifyExtAction — 告警通知动作（扩展版）
// ─────────────────────────────────────────────

// V13 架构边界说明：
//   - 本节点是 ext 版本，需要外部依赖 AlarmStorage
//   - 生成告警记录并写入存储，设置多个 Tag
//   - 适合完整告警管理场景
//
// 与 builtin 版本的区别：
//   - builtin: 简单通知回调，无外部依赖
//   - ext: 告警记录生成+存储，需要 AlarmStorage
//
// 使用场景：
//   - 完整告警管理：需要持久化告警记录
//   - 告警事件追踪：生成告警 ID 和时间戳

// AlarmStorage 是 AlarmStorage 的窄接口视图
type AlarmStorage interface {
	SaveEvent(ctx context.Context, alarmEvent any) error
}

// AlarmNotifyExtAction 生成告警记录并触发通知
type AlarmNotifyExtAction struct {
	IDValue     string
	AlarmType   string
	Severity    string
	AlarmStore  AlarmStorage
	alarmTypeOv string // 预判默认值
}

var _ core.Action = (*AlarmNotifyExtAction)(nil)

func NewAlarmNotifyExtAction(id, alarmType, severity string, store AlarmStorage) *AlarmNotifyExtAction {
	a := &AlarmNotifyExtAction{
		IDValue:    id,
		AlarmType:  alarmType,
		Severity:   severity,
		AlarmStore: store,
	}
	if a.AlarmType == "" {
		a.alarmTypeOv = "threshold"
	} else {
		a.alarmTypeOv = a.AlarmType
	}
	return a
}

func (a *AlarmNotifyExtAction) Execute(ctx context.Context, data core.DataContext) error {
	deviceID := data.DeviceID()
	pointName := data.PointName()
	value := data.Value()
	ts := data.Timestamp()

	alarmType := a.alarmTypeOv
	if at := data.GetTag("alarm_type"); at != "" {
		alarmType = at
	}

	severity := a.Severity
	if severity == "" {
		severity = "warning"
	}
	if ms := data.GetTag("alarm_severity"); ms != "" {
		severity = ms
	}

	now := time.Now()
	alarmID := fmt.Sprintf("alarm-%d-%s", now.UnixNano(), deviceID)

	// 构建告警事件数据（通过 Tag 传递，不直接依赖上层类型）
	data.SetTag("_alarm_id", alarmID)
	data.SetTag("_alarm_type", alarmType)
	data.SetTag("_alarm_severity", severity)
	data.SetTag("_alarm_notified", "true")

	// 如果有存储后端，写入告警事件
	if a.AlarmStore != nil {
		event := map[string]any{
			"id":          alarmID,
			"rule_id":     alarmType,
			"rule_name":   fmt.Sprintf("%s alarm", alarmType),
			"source_id":   deviceID,
			"source_type": "device",
			"severity":    severity,
			"status":      "pending",
			"message":     fmt.Sprintf("%s alarm: device=%s, point=%s, value=%f", alarmType, deviceID, pointName, value),
			"data": map[string]any{
				"device_id":  deviceID,
				"point_name": pointName,
				"value":      value,
				"alarm_type": alarmType,
			},
			"triggered_at": now,
		}
		_ = a.AlarmStore.SaveEvent(context.Background(), event)
	}

	// 如果 ts > 0 且没有报警时戳，补充记录
	if ts > 0 && data.GetTag("_alarm_timestamp") == "" {
		data.SetTag("_alarm_timestamp", strconv.FormatInt(ts, 10))
	}

	return nil
}

func (a *AlarmNotifyExtAction) ID() string          { return a.IDValue }
func (a *AlarmNotifyExtAction) Type() string        { return "alarm_notify_ext" }
func (a *AlarmNotifyExtAction) Description() string { return "alarm notification (ext)" }
