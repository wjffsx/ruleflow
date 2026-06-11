package otel

import (
	"context"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracerProviderAdapter 包装 otel trace.TracerProvider 为 ruleflow contract.TracerProvider。
//
// 使用方式：
//
//	import (
//	    otel "go.opentelemetry.io/otel"
//	    "github.com/wjffsx/ruleflow/contrib/otel"
//	    "github.com/wjffsx/ruleflow/core/engine"
//	)
//
//	eng := engine.NewEngine(
//	    engine.WithTracer(otel.WrapTracerProvider(otel.GetTracerProvider())),
//	)
type TracerProviderAdapter struct {
	otelTP oteltrace.TracerProvider
}

// WrapTracerProvider 将 otel trace.TracerProvider 适配为 ruleflow contract.TracerProvider。
func WrapTracerProvider(tp oteltrace.TracerProvider) contract.TracerProvider {
	if tp == nil {
		return contract.Noop()
	}
	return TracerProviderAdapter{otelTP: tp}
}

// Tracer 返回指定 name 的 Tracer
func (p TracerProviderAdapter) Tracer(name string) contract.Tracer {
	return TracerAdapter{t: p.otelTP.Tracer(name)}
}

// TracerAdapter 包装 otel trace.Tracer 为 ruleflow contract.Tracer。
type TracerAdapter struct {
	t oteltrace.Tracer
}

// Begin 启动一个 span，返回 (newCtx, end)。
// end 调用时触发 span.End()。
func (a TracerAdapter) Begin(ctx context.Context, name string) (context.Context, func()) {
	newCtx, span := a.t.Start(ctx, name)
	if span == nil {
		// otel 的 noop tracer 可能返回 nil span
		return newCtx, func() {}
	}
	return newCtx, func() { span.End() }
}

// SpanContextFromContext 从 context 中提取 otel SpanContext 并转 ruleflow 形式。
// 便捷方法：用户已有 otel 接入时，从 ctx 取 span context 后再设置到 DataContext。
func SpanContextFromContext(ctx context.Context) contract.SpanContext {
	return FromOtelSpanContext(oteltrace.SpanContextFromContext(ctx))
}
