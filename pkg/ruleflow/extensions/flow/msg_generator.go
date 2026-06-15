// Package flow provides VPP flow nodes
package flow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  MsgGeneratorAction — 定时消息生成动作
// ─────────────────────────────────────────────

// MsgGeneratorAction 定时消息生成动作
type MsgGeneratorAction struct {
	IDValue     string            `json:"id"`
	CronExpr    string            `json:"cron"`         // Cron 表达式（简化：秒/分/时）
	OutputPoint string            `json:"output_point"` // 生成的消息点名
	OutputValue float64           `json:"output_value"` // 生成的消息值
	Tags        map[string]string `json:"tags"`         // 附加标签
	IntervalSec int               `json:"interval_sec"` // 简化：间隔秒数

	// 运行时
	lastRun time.Time
	mu      sync.Mutex
}

// NewMsgGeneratorAction 创建定时消息生成动作
func NewMsgGeneratorAction(id, outputPoint string, outputValue float64, tags map[string]string, intervalSec int) *MsgGeneratorAction {
	if intervalSec == 0 {
		intervalSec = 60
	}
	return &MsgGeneratorAction{
		IDValue:     id,
		OutputPoint: outputPoint,
		OutputValue: outputValue,
		Tags:        tags,
		IntervalSec: intervalSec,
	}
}

func (a *MsgGeneratorAction) Execute(ctx context.Context, data core.DataContext) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.UnixMilli(data.Timestamp())

	// 首次执行初始化
	if a.lastRun.IsZero() {
		a.lastRun = now
		return nil
	}

	// 检查是否到达触发时间
	nextTime := a.lastRun.Add(time.Duration(a.IntervalSec) * time.Second)
	if now.Before(nextTime) {
		return nil
	}

	// 更新最后运行时间
	a.lastRun = now

	// 生成消息数据
	data.SetValue(a.OutputValue)
	if a.OutputPoint != "" {
		data.SetTag("_generated_point", a.OutputPoint)
	}
	for k, v := range a.Tags {
		data.SetTag(k, v)
	}
	data.SetTag("_generated_at", now.Format(time.RFC3339))
	data.SetTag("_msg_type", "generated")
	return nil
}

func (a *MsgGeneratorAction) ID() string   { return a.IDValue }
func (a *MsgGeneratorAction) Type() string { return "msg_generator" }
func (a *MsgGeneratorAction) Description() string {
	return fmt.Sprintf("msg generator interval=%ds", a.IntervalSec)
}
