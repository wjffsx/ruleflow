// Package slog 提供 ruleflow 引擎的 log/slog 适配器。
//
// 基本用法：
//
//	import "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/slog"
//
//	engine := core.NewEngine(core.WithLogger(slog.NewSlogLoggerDefault()))
//
// 或自定义：
//
//	custom := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
//	engine := core.NewEngine(core.WithLogger(slog.NewSlogLogger(custom)))
package slog

import (
	"context"
	"log/slog"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// SlogLogger 包装 *slog.Logger 以适配 ruleflow Logger 接口。
// Go 1.21+ 标准库自带 log/slog，无需新增依赖。
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger 构造 SlogLogger
func NewSlogLogger(l *slog.Logger) *SlogLogger {
	if l == nil {
		l = slog.Default()
	}
	return &SlogLogger{logger: l}
}

// NewSlogLoggerDefault 使用默认 slog 构造
func NewSlogLoggerDefault() *SlogLogger {
	return NewSlogLogger(slog.Default())
}

// Debug 实现 Logger
func (s *SlogLogger) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Info 实现 Logger
func (s *SlogLogger) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Warn 实现 Logger
func (s *SlogLogger) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// Error 实现 Logger
func (s *SlogLogger) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// With 返回带预设字段的子 Logger
func (s *SlogLogger) With(args ...any) contract.Logger {
	return &SlogLogger{logger: s.logger.With(args...)}
}

// ─────────────────────────────────────────────
//  slogHandler — 反向适配器（把 ruleflow.Logger 适配为 slog.Handler）
// ─────────────────────────────────────────────

// SlogFromLogger 将 ruleflow.Logger 暴露为 *slog.Logger，
// 便于希望将整个进程统一接入 ruleflow 日志体系的应用。
func SlogFromLogger(l contract.Logger) *slog.Logger {
	return slog.New(&slogBridge{inner: l})
}

type slogBridge struct {
	inner contract.Logger
}

func (b *slogBridge) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (b *slogBridge) Handle(_ context.Context, r slog.Record) error {
	args := []any{}
	r.Attrs(func(a slog.Attr) bool {
		args = append(args, a.Key, a.Value.Any())
		return true
	})
	msg := r.Message
	switch r.Level {
	case slog.LevelDebug:
		b.inner.Debug(msg, args...)
	case slog.LevelInfo:
		b.inner.Info(msg, args...)
	case slog.LevelWarn:
		b.inner.Warn(msg, args...)
	case slog.LevelError:
		b.inner.Error(msg, args...)
	}
	return nil
}

func (b *slogBridge) WithAttrs(attrs []slog.Attr) slog.Handler {
	pairs := make([]any, 0, len(attrs)*2)
	for _, a := range attrs {
		pairs = append(pairs, a.Key, a.Value.Any())
	}
	return &slogBridge{inner: b.inner.With(pairs...)}
}

func (b *slogBridge) WithGroup(_ string) slog.Handler { return b }

// 编译期接口检查
var _ contract.Logger = (*SlogLogger)(nil)
