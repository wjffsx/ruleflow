// Package slog 提供 ruleflow 引擎的 log/slog 适配器。
//
// 本文件实现 core/debug.DebugSink 接口的 slog 版本（V3.7 迁入）。
// 之前位于 core/debug/log_sink.go，命名为 LogSink；
// 为避免与同包的 SlogLogger 混淆，重命名为 DebugLogSink。
//
// 基本用法：
//
//	import "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/slog"
//
//	mgr := debug.NewDebugManager(debug.DebugOn, slog.NewDebugLogSinkDefault(), ...)
package slog

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/debug"
)

// DebugLogSink 基于 slog 的调试事件输出。
//
// V3.7：从 core/debug.LogSink 迁入；类型重命名以避免与本包的 SlogLogger 命名冲突。
// 实现 core/debug.DebugSink 接口。
type DebugLogSink struct {
	logger *slog.Logger
}

// NewDebugLogSink 构造 DebugLogSink。
// logger 为 nil 时使用 slog.Default()。
func NewDebugLogSink(logger *slog.Logger) *DebugLogSink {
	if logger == nil {
		logger = slog.Default()
	}
	return &DebugLogSink{logger: logger}
}

// NewDebugLogSinkDefault 使用 slog.Default() 构造。
func NewDebugLogSinkDefault() *DebugLogSink {
	return NewDebugLogSink(slog.Default())
}

// WriteEvent 实现 debug.DebugSink 接口。
func (s *DebugLogSink) WriteEvent(ctx context.Context, event debug.DebugEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "ruleflow.debug",
		slog.String("event", string(data)),
		slog.String("chain_id", event.ChainID),
		slog.String("rule_id", event.RuleID),
		slog.String("node_id", event.NodeID),
		slog.String("event_type", string(event.EventType)),
		slog.String("relation_type", event.RelationType),
		slog.Duration("duration", time.Duration(event.DurationNs)),
	)
	return nil
}

// 编译期接口检查
var _ debug.DebugSink = (*DebugLogSink)(nil)
