// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  质量相关条件
// ─────────────────────────────────────────────

// QualityCondition 质量条件
type QualityCondition struct {
	IDValue    string `json:"id"`
	MinQuality int    `json:"min_quality"`
}

func NewQualityCondition(id string, minQ int) *QualityCondition {
	return &QualityCondition{IDValue: id, MinQuality: minQ}
}

func (c *QualityCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	return data.Quality() >= c.MinQuality
}

func (c *QualityCondition) ID() string          { return c.IDValue }
func (c *QualityCondition) Type() string        { return "quality" }
func (c *QualityCondition) Description() string { return fmt.Sprintf("quality >= %d", c.MinQuality) }
