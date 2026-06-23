package ext

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  StatusDetectCondition — 状态检测条件
// ─────────────────────────────────────────────

// StatusDetectCondition 状态检测条件节点。
// 检测数据点值是否匹配期望的状态值，可选去抖窗口。
//
// 配置示例：
//
//	condition:
//	  leaf_type: "status_detect"
//	  leaf_config:
//	    expected_value: "offline"
//	    debounce: "5s"                # 可选：去抖窗口，5秒内不重复触发
type StatusDetectCondition struct {
	IDValue        string
	ExpectedValue  any           // 期望匹配的状态值
	ExpectedStr    string        // 字符串形式的期望值
	Debounce       time.Duration // 去抖窗口（0 表示不启用去抖）
	store          core.StateStore
}

var _ core.Condition = (*StatusDetectCondition)(nil)

func NewStatusDetectCondition(id string, expectedValue any, debounce time.Duration, store core.StateStore) *StatusDetectCondition {
	expectedStr := ""
	if s, ok := expectedValue.(string); ok {
		expectedStr = s
	}
	return &StatusDetectCondition{
		IDValue:       id,
		ExpectedValue: expectedValue,
		ExpectedStr:   expectedStr,
		Debounce:      debounce,
		store:         store,
	}
}

func (c *StatusDetectCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	val := data.Value()
	raw := data.Raw()

	// 尝试匹配
	matched := false

	// 优先从 Raw() 获取原始值进行匹配
	if raw != nil {
		if rawMap, ok := raw.(map[string]any); ok {
			if ev, ok := rawMap["value"]; ok {
				matched = c.matchValue(ev)
				if matched {
					goto matchedCheck
				}
			}
		}
	}

	// 从 DataContext Value() 匹配
	matched = c.matchFloat(val)

	if !matched {
		return false
	}

matchedCheck:
	if c.Debounce == 0 || c.store == nil {
		return true
	}

	// 去抖检查
	fqn := data.FQN()
	key := fmt.Sprintf("status_debounce:%s:%s", c.IDValue, fqn)
	now := time.Now()

	if v, ok := c.store.Get(key); ok {
		if lastTime, ok := v.(time.Time); ok {
			if now.Sub(lastTime) < c.Debounce {
				return false // 去抖窗口内，不触发
			}
		}
	}

	c.store.Set(key, now)
	return true
}

func (c *StatusDetectCondition) matchValue(v any) bool {
	switch expected := c.ExpectedValue.(type) {
	case string:
		if actual, ok := v.(string); ok {
			return actual == expected
		}
		return c.ExpectedStr == fmt.Sprintf("%v", v)
	case float64:
		if actual, ok := v.(float64); ok {
			return actual == expected
		}
	case bool:
		if actual, ok := v.(bool); ok {
			return actual == expected
		}
	}
	return false
}

func (c *StatusDetectCondition) matchFloat(v float64) bool {
	switch expected := c.ExpectedValue.(type) {
	case float64:
		return v == expected
	case string:
		return c.ExpectedStr == fmt.Sprintf("%v", v)
	}
	return false
}

func (c *StatusDetectCondition) ID() string        { return c.IDValue }
func (c *StatusDetectCondition) Type() string      { return "status_detect" }
func (c *StatusDetectCondition) Description() string {
	if c.Debounce > 0 {
		return fmt.Sprintf("status detect == %v debounce=%v", c.ExpectedValue, c.Debounce)
	}
	return fmt.Sprintf("status detect == %v", c.ExpectedValue)
}
