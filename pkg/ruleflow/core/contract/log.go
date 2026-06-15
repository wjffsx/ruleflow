// Package contract - 日志契约
package contract

import "context"

// Logger 引擎使用的结构化日志接口。
// 引擎核心不绑定具体实现（slog / zap / zerolog 等），
// 应用层可注入任意 Logger。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// LoggerContext 支持 ctx 上下文的扩展接口（可选实现）
type LoggerContext interface {
	Logger
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// NoopLogger 返回静默 Logger（默认行为）
func NoopLogger() Logger { return noopLogger{} }

type noopLogger struct{}

func (noopLogger) Debug(_ string, _ ...any) {}
func (noopLogger) Info(_ string, _ ...any)  {}
func (noopLogger) Warn(_ string, _ ...any)  {}
func (noopLogger) Error(_ string, _ ...any) {}
func (noopLogger) With(_ ...any) Logger     { return noopLogger{} }

// 编译期接口检查
var _ Logger = noopLogger{}
