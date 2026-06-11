// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  RouteAction — 路由动作
// ─────────────────────────────────────────────

// RouteAction 路由动作
type RouteAction struct {
	IDValue string   `json:"id"`
	Targets []string `json:"targets"`
}

func NewRouteAction(id string, targets []string) *RouteAction {
	return &RouteAction{IDValue: id, Targets: targets}
}

func (a *RouteAction) Execute(_ context.Context, data core.DataContext) error {
	for _, t := range a.Targets {
		data.AddTarget(t)
	}
	return nil
}

func (a *RouteAction) ID() string          { return a.IDValue }
func (a *RouteAction) Type() string        { return "route" }
func (a *RouteAction) Description() string { return "route to targets" }
