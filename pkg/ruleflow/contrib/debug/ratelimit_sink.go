// Package debug — contrib 层调试事件 sink 包装
//
// V4.9 移出：原 core/debug.DebugManager.perSecLimit 速率限制（QoS）功能迁出至本包。
// 限流是 EventBus / sink 上游包装层职责，核心管理器仅做模式过滤（Off / Failures / All）。
//
// 基本用法：
//
//	import (
//	    "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/debug"
//	    coredebug "github.com/vpptu/ruleflow/pkg/ruleflow/debug"
//	)
//
//	bus := debug.NewChannelSink(1024)
//	// 包一层限流：每秒最多 100 条
//	limitedSink := debug.NewRateLimitSink(bus, 100)
//	mgr := coredebug.NewDebugManager(coredebug.DebugAll, limitedSink, time.Time{}, 0)
package debug

import (
	"context"
	"sync/atomic"
	"time"

	coredebug "github.com/vpptu/ruleflow/pkg/ruleflow/debug"
)

// RateLimitSink 包装 sink，限制每秒事件数
//
// V4.9 移出：原 DebugManager.perSecLimit 限流能力迁入本包装。
// 应用层如需速率控制，可在 sink 链路上包一层。
type RateLimitSink struct {
	inner coredebug.DebugSink
	limit int64 // 每秒最大事件数，0 表示不限制

	curSec atomic.Int64
	curCnt atomic.Int64
}

// NewRateLimitSink 创建限流 sink 包装
//
//	inner: 被包装的 sink（事件真正写入的位置）
//	perSec: 每秒最大事件数（0 表示不限制）
func NewRateLimitSink(inner coredebug.DebugSink, perSec int) *RateLimitSink {
	if inner == nil {
		inner = coredebug.NoopSink()
	}
	return &RateLimitSink{
		inner: inner,
		limit: int64(perSec),
	}
}

// WriteEvent 实现 DebugSink 接口
func (s *RateLimitSink) WriteEvent(ctx context.Context, event coredebug.DebugEvent) error {
	if s.limit > 0 {
		now := time.Now().Unix()
		lastSec := s.curSec.Load()
		if now != lastSec {
			s.curSec.Store(now)
			s.curCnt.Store(0)
		}
		if s.curCnt.Add(1) > s.limit {
			// 超出限流，丢弃
			return nil
		}
	}
	return s.inner.WriteEvent(ctx, event)
}

// 编译期接口检查
var _ coredebug.DebugSink = (*RateLimitSink)(nil)
