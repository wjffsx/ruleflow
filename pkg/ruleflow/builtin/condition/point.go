// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"
	"regexp"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  数据点相关条件
// ─────────────────────────────────────────────

// PointNameCondition 数据点名条件
type PointNameCondition struct {
	IDValue    string              `json:"id"`
	PointNames []string            `json:"point_names"`
	nameSet    map[string]struct{} // 预编译哈希集
}

func NewPointNameCondition(id string, names []string) *PointNameCondition {
	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}
	return &PointNameCondition{IDValue: id, PointNames: names, nameSet: nameSet}
}

func (c *PointNameCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	_, ok := c.nameSet[data.PointName()] // O(1) 查找
	return ok
}

func (c *PointNameCondition) ID() string   { return c.IDValue }
func (c *PointNameCondition) Type() string { return "point_name" }
func (c *PointNameCondition) Description() string {
	return fmt.Sprintf("point name in %v", c.PointNames)
}

// PointNamePatternCondition 数据点名模式条件
type PointNamePatternCondition struct {
	IDValue string         `json:"id"`
	Pattern string         `json:"pattern"`
	regex   *regexp.Regexp // 预编译正则
}

func NewPointNamePatternCondition(id, pattern string) (*PointNamePatternCondition, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	return &PointNamePatternCondition{IDValue: id, Pattern: pattern, regex: re}, nil
}

func (c *PointNamePatternCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	return c.regex.MatchString(data.PointName())
}

func (c *PointNamePatternCondition) ID() string   { return c.IDValue }
func (c *PointNamePatternCondition) Type() string { return "point_name_pattern" }
func (c *PointNamePatternCondition) Description() string {
	return fmt.Sprintf("point name matches %s", c.Pattern)
}

// FQNCondition FQN 前缀条件
// 使用 Trie 前缀树替代线性扫描，O(K) 匹配（K = FQN 长度）
type FQNCondition struct {
	IDValue  string      `json:"id"`
	Prefixes []string    `json:"prefixes"`
	trie     *prefixTrie // 编译时构建
}

// prefixTrie 前缀 Trie 树
type prefixTrie struct {
	children map[byte]*prefixTrie
	isEnd    bool
}

func newPrefixTrie(prefixes []string) *prefixTrie {
	root := &prefixTrie{children: make(map[byte]*prefixTrie)}
	for _, p := range prefixes {
		node := root
		for i := 0; i < len(p); i++ {
			if node.children == nil {
				node.children = make(map[byte]*prefixTrie)
			}
			child, ok := node.children[p[i]]
			if !ok {
				child = &prefixTrie{}
				node.children[p[i]] = child
			}
			node = child
		}
		node.isEnd = true
	}
	return root
}

func (t *prefixTrie) match(s string) bool {
	node := t
	for i := 0; i < len(s); i++ {
		if node.isEnd {
			return true // 前缀匹配成功
		}
		child, ok := node.children[s[i]]
		if !ok {
			return false
		}
		node = child
	}
	return node.isEnd
}

func NewFQNCondition(id string, prefixes []string) *FQNCondition {
	return &FQNCondition{
		IDValue:  id,
		Prefixes: prefixes,
		trie:     newPrefixTrie(prefixes),
	}
}

func (c *FQNCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	return c.trie.match(data.FQN())
}

func (c *FQNCondition) ID() string          { return c.IDValue }
func (c *FQNCondition) Type() string        { return "fqn_prefix" }
func (c *FQNCondition) Description() string { return fmt.Sprintf("FQN prefix in %v", c.Prefixes) }
