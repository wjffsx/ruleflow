// Package condition provides builtin condition nodes
package condition

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  越限相关条件
// ─────────────────────────────────────────────

// LimitExceededCondition 越限条件
type LimitExceededCondition struct {
	IDValue string `json:"id"`
}

func NewLimitExceededCondition(id string) *LimitExceededCondition {
	return &LimitExceededCondition{IDValue: id}
}

func (c *LimitExceededCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	return data.LimitExceeded()
}

func (c *LimitExceededCondition) ID() string          { return c.IDValue }
func (c *LimitExceededCondition) Type() string        { return "limit_exceeded" }
func (c *LimitExceededCondition) Description() string { return "limit exceeded" }
