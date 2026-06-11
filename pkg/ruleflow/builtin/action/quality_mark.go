// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  QualityMarkAction — 质量标记动作（简单版）
// ─────────────────────────────────────────────

// QualityMarkAction 质量标记动作（简单版）
//
// V13 架构边界说明：
//   - 本节点是 builtin 版本，无外部依赖
//   - 根据 LimitExceeded() 状态设置质量值
//   - 适合简单场景，不需要存储同步
//
// 与 ext 版本的区别：
//   - builtin: 根据越限状态设置质量，无外部依赖
//   - ext: 根据配置/Tag 设置质量，并同步写存储
//
// 使用场景：
//   - 简单质量标记：根据越限状态自动设置质量
//   - 无存储需求：不需要同步写存储
type QualityMarkAction struct {
	IDValue   string `json:"id"`
	GoodValue int    `json:"good_value"`
	BadValue  int    `json:"bad_value"`
}

func NewQualityMarkAction(id string, good, bad int) *QualityMarkAction {
	return &QualityMarkAction{IDValue: id, GoodValue: good, BadValue: bad}
}

func (a *QualityMarkAction) Execute(_ context.Context, data core.DataContext) error {
	if data.LimitExceeded() {
		data.SetQuality(a.BadValue)
	} else {
		data.SetQuality(a.GoodValue)
	}
	return nil
}

func (a *QualityMarkAction) ID() string          { return a.IDValue }
func (a *QualityMarkAction) Type() string        { return "quality_mark" }
func (a *QualityMarkAction) Description() string { return "mark quality based on limit check" }
