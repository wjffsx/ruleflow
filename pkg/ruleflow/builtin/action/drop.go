// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DropAction — 丢弃动作
// ─────────────────────────────────────────────

// DropAction 丢弃动作
type DropAction struct {
	IDValue string `json:"id"`
}

func NewDropAction(id string) *DropAction {
	return &DropAction{IDValue: id}
}

func (a *DropAction) Execute(_ context.Context, data core.DataContext) error {
	return core.ErrDropData
}

func (a *DropAction) ID() string          { return a.IDValue }
func (a *DropAction) Type() string        { return "drop" }
func (a *DropAction) Description() string { return "drop data point" }
