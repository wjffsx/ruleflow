package ext

import (
	"context"
	"fmt"
	"strings"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  EmitSOEAction — SOE事件生成动作
// ─────────────────────────────────────────────

// SOEPublisher SOE事件发布接口（由服务层实现）
type SOEPublisher interface {
	PublishSOE(deviceID, pointName string, eventType string, value float64, ts int64, severity string, desc string) error
}

// EmitSOEAction SOE事件生成动作
type EmitSOEAction struct {
	IDValue             string
	EventType           string // 事件类型
	Severity            string // 严重程度 "info" / "warning" / "critical"
	DescriptionTemplate string // 描述模板
	Publisher           SOEPublisher // 外部注入
}

var _ core.Action = (*EmitSOEAction)(nil)

// NewEmitSOEAction 创建SOE事件生成动作
func NewEmitSOEAction(id, eventType, severity, descriptionTemplate string, publisher SOEPublisher) *EmitSOEAction {
	if severity == "" {
		severity = "info"
	}
	return &EmitSOEAction{
		IDValue:             id,
		EventType:           eventType,
		Severity:            severity,
		DescriptionTemplate: descriptionTemplate,
		Publisher:           publisher,
	}
}

func (a *EmitSOEAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Publisher == nil {
		return nil // 无发布器，跳过
	}

	desc := a.renderTemplate(data)
	return a.Publisher.PublishSOE(
		data.DeviceID(),
		data.PointName(),
		a.EventType,
		data.Value(),
		data.Timestamp(),
		a.Severity,
		desc,
	)
}

// renderTemplate 渲染描述模板
func (a *EmitSOEAction) renderTemplate(data core.DataContext) string {
	desc := a.DescriptionTemplate
	if desc == "" {
		desc = "{device_id}/{point_name} value={value}"
	}

	// 简单模板替换
	desc = strings.ReplaceAll(desc, "{device_id}", data.DeviceID())
	desc = strings.ReplaceAll(desc, "{point_name}", data.PointName())
	desc = strings.ReplaceAll(desc, "{value}", fmt.Sprintf("%.2f", data.Value()))
	desc = strings.ReplaceAll(desc, "{timestamp}", fmt.Sprintf("%d", data.Timestamp()))
	desc = strings.ReplaceAll(desc, "{quality}", fmt.Sprintf("%d", data.Quality()))

	return desc
}

func (a *EmitSOEAction) ID() string          { return a.IDValue }
func (a *EmitSOEAction) Type() string        { return "emit_soe" }
func (a *EmitSOEAction) Description() string { return fmt.Sprintf("emit SOE event %s", a.EventType) }