// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  LimitCheckAction — 越限检测动作
// ─────────────────────────────────────────────

// LimitCheckAction 越限检测动作
type LimitCheckAction struct {
	IDValue string `json:"id"`
}

func NewLimitCheckAction(id string) *LimitCheckAction {
	return &LimitCheckAction{IDValue: id}
}

func (a *LimitCheckAction) Execute(_ context.Context, data core.DataContext) error {
	v := data.Value()
	exceeded := false
	if upper, ok := data.UpperLimit(); ok && v > upper {
		exceeded = true
	}
	if lower, ok := data.LowerLimit(); ok && v < lower {
		exceeded = true
	}
	data.SetLimitExceeded(exceeded)
	return nil
}

func (a *LimitCheckAction) ID() string          { return a.IDValue }
func (a *LimitCheckAction) Type() string        { return "limit_check" }
func (a *LimitCheckAction) Description() string { return "check value against limits" }
