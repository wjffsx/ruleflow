// Package debug 提供 ruleflow 引擎的调试事件 sink 包装层。
//
// V4.10 重命名：原 EventBus 重命名为 ChannelSink（语义清晰化：实为"带缓冲的非阻塞 channel sink"，
// 与 pub/sub 概念无关；之前的 EventBus 命名易引起"事件总线/多生产者多消费者"误解）。
//
// 基本架构：
//
//	ruleflow Engine → DebugSink → ChannelSink (chan) → gRPC Server-Side Streaming
//	                                                  → SSE fallback
//
// ChannelSink is the central bridge: ruleflow writes events into it (zero-blocking),
// and all clients consume from it via independent goroutines.
package debug

import (
	"sync/atomic"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/debug"
)

// ChannelSink 带缓冲的非阻塞 channel sink（V4.10 替代 EventBus）
type ChannelSink struct {
	ch      chan debug.DebugEvent
	dropped atomic.Int64 // 因 buffer 满丢弃的计数
	sent    atomic.Int64 // 成功入队计数
}

// NewChannelSink 创建带缓冲的 channel sink
//   - bufferSize: 缓冲区大小（建议 1024-4096）
func NewChannelSink(bufferSize int) *ChannelSink {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	return &ChannelSink{
		ch: make(chan debug.DebugEvent, bufferSize),
	}
}

// WriteEvent 实现 debug.DebugSink 接口（引擎热路径调用）
//
// 非阻塞写入：buffer 满时丢弃事件，绝不阻塞引擎评估热路径。
func (b *ChannelSink) WriteEvent(ctx any, event debug.DebugEvent) error {
	select {
	case b.ch <- event:
		b.sent.Add(1)
		return nil
	default:
		// buffer 满，丢弃事件（绝不阻塞引擎）
		b.dropped.Add(1)
		return nil
	}
}

// Subscribe 返回只读 channel 供 gRPC/SSE 消费
func (b *ChannelSink) Subscribe() <-chan debug.DebugEvent {
	return b.ch
}

// Stats 返回统计信息
func (b *ChannelSink) Stats() (sent, dropped int64) {
	return b.sent.Load(), b.dropped.Load()
}

// Close 关闭 channel sink，停止所有消费者
func (b *ChannelSink) Close() {
	// safe to close a chan only once; use sync.Once if multiple callers
	close(b.ch)
}

// compile-time check: *ChannelSink implements debug.DebugSink
var _ debug.DebugSink = (*ChannelSink)(nil)
