package ext

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  AlarmEmitAction — 告警事件发射动作
// ─────────────────────────────────────────────

// AlarmEmitAction 告警事件发射动作。
// 在 DataContext 上设置告警相关的 Tag，供后续阶段的 AlarmManager 消费。
// 与 alarm_notify_ext 的区别：不依赖外部存储，仅做告警标记，适合 Pipeline 内的轻量告警。
//
// 配置示例：
//
//	actions:
//	  - type: "alarm_emit"
//	    config:
//	      severity: "warning"         # info / warning / error / critical
//	      source_type: "device"       # device / system
//	      message_template: "{{.device}} {{.point}} value={{.value}} exceeded"
type AlarmEmitAction struct {
	IDValue         string
	Severity        string // info / warning / error / critical
	SourceType      string // device / system
	MessageTemplate string // 可选：消息模板
}

var _ core.Action = (*AlarmEmitAction)(nil)

func NewAlarmEmitAction(id, severity, sourceType, messageTemplate string) *AlarmEmitAction {
	if severity == "" {
		severity = "warning"
	}
	if sourceType == "" {
		sourceType = "device"
	}
	return &AlarmEmitAction{
		IDValue:         id,
		Severity:        severity,
		SourceType:      sourceType,
		MessageTemplate: messageTemplate,
	}
}

func (a *AlarmEmitAction) Execute(_ context.Context, data core.DataContext) error {
	deviceID := data.DeviceID()
	pointName := data.PointName()
	value := data.Value()
	now := time.Now()

	alarmID := fmt.Sprintf("alarm-%d-%s", now.UnixNano(), deviceID)

	// 从 Tag 获取覆盖值（允许上游节点动态设置）
	severity := a.Severity
	if s := data.GetTag("alarm_severity"); s != "" {
		severity = s
	}

	sourceType := a.SourceType
	if s := data.GetTag("alarm_source_type"); s != "" {
		sourceType = s
	}

	// 构建告警消息
	message := fmt.Sprintf("%s alarm: device=%s, point=%s, value=%v", severity, deviceID, pointName, value)
	if a.MessageTemplate != "" {
		message = a.MessageTemplate
		// 简单模板替换
		templateVars := map[string]string{
			"{{.device}}":   deviceID,
			"{{.point}}":    pointName,
			"{{.value}}":    fmt.Sprintf("%v", value),
			"{{.severity}}": severity,
		}
		for k, v := range templateVars {
			// 简单字符串替换，非完整 Go template
			for i := 0; i < 3; i++ {
				newMsg := ""
				for pos := 0; pos < len(message); pos++ {
					if pos+len(k) <= len(message) && message[pos:pos+len(k)] == k {
						newMsg += v
						pos += len(k) - 1
					} else {
						newMsg += string(message[pos])
					}
				}
				message = newMsg
			}
		}
	}

	// 设置告警 Tags
	data.SetTag("_alarm_id", alarmID)
	data.SetTag("_alarm_severity", severity)
	data.SetTag("_alarm_source_type", sourceType)
	data.SetTag("_alarm_message", message)
	data.SetTag("_alarm_timestamp", fmt.Sprintf("%d", now.UnixMilli()))
	data.SetTag("_alarm_notified", "true")

	// 可选：通过 upstream Tag 传递告警规则 ID 和类型
	if ruleID := data.GetTag("_rule_id"); ruleID != "" {
		data.SetTag("_alarm_rule_id", ruleID)
	}
	if alarmType := data.GetTag("_alarm_type"); alarmType == "" {
		// 如果是阈值检测触发的告警，设置默认类型
		if data.LimitExceeded() || data.GetTag("threshold_operator") != "" {
			data.SetTag("_alarm_type", "threshold")
		} else {
			data.SetTag("_alarm_type", "detection")
		}
	}

	return nil
}

func (a *AlarmEmitAction) ID() string        { return a.IDValue }
func (a *AlarmEmitAction) Type() string      { return "alarm_emit" }
func (a *AlarmEmitAction) Description() string {
	return fmt.Sprintf("emit alarm: severity=%s, source=%s", a.Severity, a.SourceType)
}
