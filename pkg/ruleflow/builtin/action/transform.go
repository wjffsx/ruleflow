// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  TransformAction — 值变换动作
// ─────────────────────────────────────────────

// TransformAction 值变换动作（scale + offset）
type TransformAction struct {
	IDValue   string   `json:"id"`
	Scale     *float64 `json:"scale"`
	Offset    *float64 `json:"offset"`
	Unit      string   `json:"unit"`
	hasScale  bool
	hasOffset bool
	scaleVal  float64
	offsetVal float64
}

func NewTransformAction(id string, scale, offset *float64, unit string) *TransformAction {
	a := &TransformAction{IDValue: id, Unit: unit}
	if scale != nil {
		a.hasScale = true
		a.scaleVal = *scale
	}
	if offset != nil {
		a.hasOffset = true
		a.offsetVal = *offset
	}
	return a
}

func (a *TransformAction) Execute(_ context.Context, data core.DataContext) error {
	v := data.Value()
	if a.hasScale {
		v = v * a.scaleVal
	}
	if a.hasOffset {
		v = v + a.offsetVal
	}
	data.SetValue(v)
	return nil
}

func (a *TransformAction) ID() string          { return a.IDValue }
func (a *TransformAction) Type() string        { return "transform" }
func (a *TransformAction) Description() string { return "transform value (scale + offset)" }
