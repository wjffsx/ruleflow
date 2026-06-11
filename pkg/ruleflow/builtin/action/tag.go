// Package action provides builtin action nodes
package action

import (
	"context"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  TagAction — 标签动作
// ─────────────────────────────────────────────

// TagAction 标签动作
type TagAction struct {
	IDValue string            `json:"id"`
	Tags    map[string]string `json:"tags"`
}

func NewTagAction(id string, tags map[string]string) *TagAction {
	return &TagAction{IDValue: id, Tags: tags}
}

func (a *TagAction) Execute(_ context.Context, data core.DataContext) error {
	for k, v := range a.Tags {
		data.SetTag(k, v)
	}
	return nil
}

func (a *TagAction) ID() string          { return a.IDValue }
func (a *TagAction) Type() string        { return "tag" }
func (a *TagAction) Description() string { return "add tags" }
