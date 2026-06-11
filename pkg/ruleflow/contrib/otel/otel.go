// Package otel 提供 ruleflow core/contract 与 go.opentelemetry.io/otel 之间的双向适配。
//
// V2 迁移：原 core.DataContext.SpanContext() 返回 otel.SpanContext，
// V2 改为返回 contract.SpanContext（零 otel 依赖）。
// 如果你的应用已经使用 otel 进行追踪，可使用本包提供的适配函数：
//
//	import (
//	    "go.opentelemetry.io/otel/trace"
//	    "github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
//	    "github.com/vpptu/ruleflow/contrib/otel"
//	)
//
//	// otel.SpanContext → contract.SpanContext
//	sc := otel.FromOtelSpanContext(otelSc)
//	dc.SetSpanContext(sc)
//	// contract.SpanContext → otel.SpanContext
//	otelSc = otel.ToOtelSpanContext(sc)
//	trace.SpanContextFromContext(ctx)
//
// 适用场景：
//   - 应用层使用 otel 作为底层追踪后端
//   - 已有 otel 接入，希望最小改动接入 ruleflow
//
// 本包不强制引入 otel：仅在需要时 import。
package otel

import (
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// ToOtelSpanContext 将 contract.SpanContext 转换为 otel trace.SpanContext。
//
// 字段映射：
//   - TraceID   → TraceID
//   - SpanID    → SpanID
//   - TraceFlags → TraceFlags
//   - TraceState → TraceState
func ToOtelSpanContext(sc contract.SpanContext) oteltrace.SpanContext {
	if sc.IsZero() {
		return oteltrace.SpanContext{}
	}
	out := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID(sc.TraceID),
		SpanID:     oteltrace.SpanID(sc.SpanID),
		TraceFlags: oteltrace.TraceFlags(sc.TraceFlags),
		Remote:     false,
	})
	if len(sc.TraceState.List) > 0 {
		var ts oteltrace.TraceState
		for _, e := range sc.TraceState.List {
			ts, _ = ts.Insert(e.Key, e.Value)
		}
		out = out.WithTraceState(ts)
	}
	return out
}

// FromOtelSpanContext 将 otel trace.SpanContext 转换为 contract.SpanContext。
//
// 字段映射：
//   - TraceID   → TraceID
//   - SpanID    → SpanID
//   - TraceFlags → TraceFlags
//   - TraceState → TraceState
func FromOtelSpanContext(sc oteltrace.SpanContext) contract.SpanContext {
	if !sc.IsValid() {
		return contract.SpanContext{}
	}
	out := contract.SpanContext{
		TraceID:    contract.TraceID(sc.TraceID()),
		SpanID:     contract.SpanID(sc.SpanID()),
		TraceFlags: byte(sc.TraceFlags()),
	}
	ts := sc.TraceState()
	if ts.Len() > 0 {
		ts.Walk(func(k, v string) bool {
			out.TraceState.List = append(out.TraceState.List, contract.TraceStateEntry{Key: k, Value: v})
			return true
		})
	}
	return out
}
