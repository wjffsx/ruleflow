// Package condition provides builtin condition nodes
package condition

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  时间窗口条件
// ─────────────────────────────────────────────

// TimeWindowCondition 判断 DataContext.Timestamp 是否在指定时间窗口内
type TimeWindowCondition struct {
	IDValue  string
	Start    string // "08:00:00"
	End      string // "18:00:00"
	Timezone string // "Asia/Shanghai"
	Weekdays []int  // 0=Sunday ~ 6=Saturday，空=不限
}

func NewTimeWindowCondition(id, start, end, timezone string, weekdays []int) *TimeWindowCondition {
	return &TimeWindowCondition{
		IDValue:  id,
		Start:    start,
		End:      end,
		Timezone: timezone,
		Weekdays: weekdays,
	}
}

func (c *TimeWindowCondition) Evaluate(_ context.Context, data core.DataContext) bool {
	ts := time.Unix(0, data.Timestamp())
	if data.Timestamp() > 1e18 {
		// 毫秒级时间戳
		ts = time.UnixMilli(data.Timestamp())
	}

	loc := time.UTC
	if c.Timezone != "" {
		if l, err := time.LoadLocation(c.Timezone); err == nil {
			loc = l
		}
	}
	local := ts.In(loc)

	// 星期检查
	if len(c.Weekdays) > 0 {
		wd := int(local.Weekday())
		found := false
		for _, d := range c.Weekdays {
			if d == wd {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 时间窗口检查
	startSec := parseTimeOfDay(c.Start)
	endSec := parseTimeOfDay(c.End)
	currentSec := local.Hour()*3600 + local.Minute()*60 + local.Second()

	if startSec < 0 || endSec < 0 {
		return false
	}

	// 支持跨午夜（如 22:00-06:00）
	if startSec <= endSec {
		return currentSec >= startSec && currentSec <= endSec
	}
	// 跨午夜
	return currentSec >= startSec || currentSec <= endSec
}

func (c *TimeWindowCondition) ID() string   { return c.IDValue }
func (c *TimeWindowCondition) Type() string { return "time_window" }
func (c *TimeWindowCondition) Description() string {
	return fmt.Sprintf("time window %s-%s %s", c.Start, c.End, c.Timezone)
}

// parseTimeOfDay 将 "HH:MM:SS" 解析为当天的秒数
func parseTimeOfDay(s string) int {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return -1
	}
	h := atoi(parts[0])
	m := atoi(parts[1])
	sec := 0
	if len(parts) > 2 {
		sec = atoi(parts[2])
	}
	if h < 0 || h > 23 || m < 0 || m > 59 || sec < 0 || sec > 59 {
		return -1
	}
	return h*3600 + m*60 + sec
}

func atoi(s string) int {
	v := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		v = v*10 + int(c-'0')
	}
	return v
}
