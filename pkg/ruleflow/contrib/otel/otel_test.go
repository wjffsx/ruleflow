package otel

import (
	"testing"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestToOtelSpanContext_Zero(t *testing.T) {
	out := ToOtelSpanContext(contract.SpanContext{})
	if out.IsValid() {
		t.Error("zero input should produce invalid otel SpanContext")
	}
}

func TestFromOtelSpanContext_Invalid(t *testing.T) {
	out := FromOtelSpanContext(oteltrace.SpanContext{})
	if !out.IsZero() {
		t.Error("invalid otel SpanContext should produce zero contract.SpanContext")
	}
}

func TestRoundTrip(t *testing.T) {
	original := contract.SpanContext{
		TraceID:    contract.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     contract.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: 0x01,
	}
	otelSc := ToOtelSpanContext(original)
	if !otelSc.IsSampled() {
		t.Error("otel SpanContext should be sampled")
	}
	round := FromOtelSpanContext(otelSc)
	if round.TraceID != original.TraceID {
		t.Errorf("TraceID roundtrip mismatch")
	}
	if round.SpanID != original.SpanID {
		t.Errorf("SpanID roundtrip mismatch")
	}
	if round.TraceFlags != original.TraceFlags {
		t.Errorf("TraceFlags roundtrip mismatch")
	}
}
