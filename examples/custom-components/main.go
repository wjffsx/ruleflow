// Package main 演示自定义 Condition / Action 的注册与使用。
//
// 场景：在规则链中注入应用层特定的领域逻辑。
// 这里示例：
//   - 自定义 Condition：判断设备是否在维护窗口内
//   - 自定义 Action：将超限事件推送到外部 HTTP 端点
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/engine"
	"github.com/vpptu/ruleflow/pkg/ruleflow/datacontext"
)

// ─────────────────────────────────────────────
//  自定义 Condition
// ─────────────────────────────────────────────

// MaintenanceWindowCondition 维护窗口条件：仅在指定时间段内评估
type MaintenanceWindowCondition struct {
	id    string
	start int // hour
	end   int
}

func (m *MaintenanceWindowCondition) ID() string   { return m.id }
func (m *MaintenanceWindowCondition) Type() string { return "maintenance_window" }
func (m *MaintenanceWindowCondition) Description() string {
	return fmt.Sprintf("maintenance window [%02d:00,%02d:00)", m.start, m.end)
}
func (m *MaintenanceWindowCondition) Evaluate(_ context.Context, _ core.DataContext) bool {
	h := time.Now().Hour()
	if m.start < m.end {
		return h >= m.start && h < m.end
	}
	// 跨午夜窗口（如 22:00-06:00）
	return h >= m.start || h < m.end
}

// ─────────────────────────────────────────────
//  自定义 Action
// ─────────────────────────────────────────────

// WebhookAction 推送事件到外部 HTTP 端点（示例只打印，不实际发请求）
type WebhookAction struct {
	id     string
	URL    string
	Events []string
}

func (w *WebhookAction) ID() string   { return w.id }
func (w *WebhookAction) Type() string { return "webhook" }
func (w *WebhookAction) Description() string {
	return fmt.Sprintf("POST to %s on events %v", w.URL, w.Events)
}
func (w *WebhookAction) Execute(_ context.Context, data core.DataContext) error {
	fmt.Printf("[webhook %s] -> %s  value=%v  quality=%d  fqn=%s\n",
		w.URL, w.Events, data.Value(), data.Quality(), data.FQN())
	return nil
}

// ─────────────────────────────────────────────
//  使用示例：手动构造 RuleChain
// ─────────────────────────────────────────────

func main() {
	chain := &core.RuleChain{
		ID:   "custom_chain",
		Name: "custom components demo",
		Root: true,
		Rules: []*core.Rule{{
			ID:       "maintenance-alarm",
			Priority: 1,
			Enabled:  true,
			Condition: &core.ConditionNode{
				Leaf: &MaintenanceWindowCondition{
					id:    "maintenance-window",
					start: 22,
					end:   6,
				},
			},
			Actions: &core.ActionChain{
				Actions: []core.Action{
					&WebhookAction{
						id:     "send-webhook",
						URL:    "https://ops.example.com/hooks/alarm",
						Events: []string{"over_limit", "quality_bad"},
					},
				},
			},
		}},
	}

	e := engine.NewEngine()
	if err := e.LoadChain(chain); err != nil {
		panic(err)
	}

	data := datacontext.NewMapDataContext(map[string]any{
		"device_id":  "sensor-42",
		"point_name": "temperature",
		"value":      85.5,
		"quality":    192,
	})
	result, err := e.EvalChain(context.Background(), "custom_chain", data)
	if err != nil {
		fmt.Printf("eval err: %v\n", err)
		return
	}
	fmt.Printf("matched %d rule(s)\n", len(result.MatchedRules))
}
