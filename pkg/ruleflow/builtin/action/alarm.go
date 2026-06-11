// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  AlarmNotifyAction — 告警通知动作（简单版）
// ─────────────────────────────────────────────

// AlarmNotifyAction 告警通知动作（简单版）
//
// V13 架构边界说明：
//   - 本节点是 builtin 版本，无外部依赖
//   - 通过 NotifyFunc 回调函数发送通知
//   - 适合简单场景，不需要告警存储
//
// 与 ext 版本的区别：
//   - builtin: 简单通知回调，无外部依赖
//   - ext: 告警记录生成+存储，需要 AlarmStorage
//
// 使用场景：
//   - 简单告警通知：通过回调发送短信/邮件/推送
//   - 无存储需求：不需要持久化告警记录
type AlarmNotifyAction struct {
	IDValue    string                                              `json:"id"`
	Severity   string                                              `json:"severity"` // info / warning / critical
	Message    string                                              `json:"message"`
	NotifyFunc func(deviceID, pointName, severity, message string) `json:"-"`
}

func NewAlarmNotifyAction(id, severity, message string, notifyFunc func(string, string, string, string)) *AlarmNotifyAction {
	return &AlarmNotifyAction{
		IDValue:    id,
		Severity:   severity,
		Message:    message,
		NotifyFunc: notifyFunc,
	}
}

func (a *AlarmNotifyAction) Execute(_ context.Context, data core.DataContext) error {
	if a.NotifyFunc != nil {
		a.NotifyFunc(data.DeviceID(), data.PointName(), a.Severity, a.Message)
	}
	return nil
}

func (a *AlarmNotifyAction) ID() string          { return a.IDValue }
func (a *AlarmNotifyAction) Type() string        { return "alarm_notify" }
func (a *AlarmNotifyAction) Description() string { return "send alarm notification" }
