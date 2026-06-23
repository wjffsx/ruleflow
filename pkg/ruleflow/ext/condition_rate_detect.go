package ext

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  RateDetectCondition — 变化率检测条件
// ─────────────────────────────────────────────

// RateDetectCondition 变化率检测条件节点。
// 检测数据点值在时间窗口内的变化率是否超过阈值。
//
// 配置示例：
//
//	condition:
//	  leaf_type: "rate_detect"
//	  leaf_config:
//	    max_rate: 100               # 最大变化率（单位/秒）
//	    window: "60s"               # 检测窗口
//	    direction: "both"           # both/rise/fall
type RateDetectCondition struct {
	IDValue   string
	MaxRate   float64       // 最大变化率阈值
	Window    time.Duration // 检测窗口
	Direction string        // both / rise / fall
	store     core.StateStore
}

var _ core.Condition = (*RateDetectCondition)(nil)

func NewRateDetectCondition(id string, maxRate float64, window time.Duration, direction string, store core.StateStore) *RateDetectCondition {
	if direction == "" {
		direction = "both"
	}
	return &RateDetectCondition{
		IDValue:   id,
		MaxRate:   maxRate,
		Window:    window,
		Direction: direction,
		store:     store,
	}
}

type rateEntry struct {
	value     float64
	timestamp time.Time
}

func (c *RateDetectCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	if c.store == nil || c.MaxRate <= 0 {
		return false
	}

	fqn := data.FQN()
	key := fmt.Sprintf("rate_detect:%s:%s", c.IDValue, fqn)
	now := time.Now()
	val := data.Value()

	// 获取或创建窗口记录
	var entries []rateEntry
	if v, ok := c.store.Get(key); ok {
		if e, ok := v.([]rateEntry); ok {
			entries = e
		}
	}

	// 添加当前值
	entries = append(entries, rateEntry{value: val, timestamp: now})

	// 裁剪窗口：只保留窗口内的记录
	var filtered []rateEntry
	cutoff := now.Add(-c.Window)
	for _, e := range entries {
		if e.timestamp.After(cutoff) {
			filtered = append(filtered, e)
		}
	}
	entries = filtered

	// 更新存储
	c.store.Set(key, entries)

	// 需要至少 2 个点才能计算变化率
	if len(entries) < 2 {
		return false
	}

	// 计算窗口内总变化率
	first := entries[0]
	last := entries[len(entries)-1]
	elapsed := last.timestamp.Sub(first.timestamp).Seconds()
	if elapsed <= 0 {
		return false
	}

	delta := last.value - first.value
	rate := math.Abs(delta) / elapsed

	exceeded := rate > c.MaxRate
	if !exceeded {
		return false
	}

	// 根据方向过滤
	if c.Direction == "rise" && delta <= 0 {
		return false
	}
	if c.Direction == "fall" && delta >= 0 {
		return false
	}

	return true
}

func (c *RateDetectCondition) ID() string        { return c.IDValue }
func (c *RateDetectCondition) Type() string      { return "rate_detect" }
func (c *RateDetectCondition) Description() string {
	return fmt.Sprintf("rate detect max=%.2f/s window=%v direction=%s", c.MaxRate, c.Window, c.Direction)
}
