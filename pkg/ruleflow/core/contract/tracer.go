// Package contract - 追踪器契约
package contract

import "context"

// BeginFunc 启动一个 span
type BeginFunc func(ctx context.Context, name string) (newCtx context.Context, end func())

// Tracer 抽象的追踪器接口
type Tracer interface {
	Begin(ctx context.Context, name string) (newCtx context.Context, end func())
}

// TracerProvider 提供命名 tracer
type TracerProvider interface {
	Tracer(name string) Tracer
}

// Noop 返回无操作 TracerProvider
func Noop() TracerProvider { return noopProvider{} }

type noopTracer struct{}

func (noopTracer) Begin(ctx context.Context, _ string) (context.Context, func()) {
	return ctx, func() {}
}

type noopProvider struct{}

func (noopProvider) Tracer(_ string) Tracer { return noopTracer{} }

// 编译期接口检查
var _ Tracer = noopTracer{}
var _ TracerProvider = noopProvider{}

// FuncProvider 适配器：将 BeginFunc 包装为 TracerProvider
type FuncProvider struct {
	name  string
	begin BeginFunc
}

// FuncTracerProvider 创建 TracerProvider
func FuncTracerProvider(name string, begin BeginFunc) TracerProvider {
	if begin == nil {
		return noopProvider{}
	}
	return FuncProvider{name: name, begin: begin}
}

// Tracer 返回指定 name 的 Tracer
func (p FuncProvider) Tracer(name string) Tracer {
	if p.begin == nil {
		return noopTracer{}
	}
	_ = name
	return funcTracer{begin: p.begin}
}

type funcTracer struct {
	begin BeginFunc
}

func (f funcTracer) Begin(ctx context.Context, name string) (context.Context, func()) {
	return f.begin(ctx, name)
}
