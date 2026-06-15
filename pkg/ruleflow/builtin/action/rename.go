// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  RenameAction — 重命名动作
// ─────────────────────────────────────────────

// RenameAction 重命名动作
type RenameAction struct {
	IDValue   string `json:"id"`
	PointName string `json:"point_name"`
}

func NewRenameAction(id, pointName string) *RenameAction {
	return &RenameAction{IDValue: id, PointName: pointName}
}

func (a *RenameAction) Execute(_ context.Context, data core.DataContext) error {
	// 直接设置新名称（DataContext 接口已支持 SetPointName）
	data.SetPointName(a.PointName)
	return nil
}

func (a *RenameAction) ID() string          { return a.IDValue }
func (a *RenameAction) Type() string        { return "rename" }
func (a *RenameAction) Description() string { return "rename point" }
