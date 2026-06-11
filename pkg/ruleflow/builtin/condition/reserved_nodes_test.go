// Package condition_test provides tests for builtin condition nodes
package condition_test

import (
	"context"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// mockDataContext 用于测试的模拟 DataContext
type mockDataContext struct {
	deviceID     string
	pointName    string
	value        float64
	quality      int
	prevValue    float64
	hasPrevValue bool
	tags         map[string]string
}

func newMockData() *mockDataContext {
	return &mockDataContext{
		deviceID:  "device001",
		pointName: "power",
		tags:      make(map[string]string),
	}
}

func (m *mockDataContext) DeviceID() string           { return m.deviceID }
func (m *mockDataContext) PointName() string          { return m.pointName }
func (m *mockDataContext) PointType() string          { return "analog" }
func (m *mockDataContext) FQN() string                { return m.deviceID + "/" + m.pointName }
func (m *mockDataContext) Value() float64             { return m.value }
func (m *mockDataContext) SetValue(v float64)         { m.value = v }
func (m *mockDataContext) Quality() int               { return m.quality }
func (m *mockDataContext) SetQuality(q int)           { m.quality = q }
func (m *mockDataContext) UpperLimit() (float64, bool) { return 100.0, true }
func (m *mockDataContext) LowerLimit() (float64, bool) { return 0.0, true }
func (m *mockDataContext) LimitExceeded() bool        { return false }
func (m *mockDataContext) SetLimitExceeded(v bool)    {}
func (m *mockDataContext) GetTag(key string) string   { return m.tags[key] }
func (m *mockDataContext) SetTag(key, value string)   { m.tags[key] = value }
func (m *mockDataContext) TargetCount() int           { return 0 }
func (m *mockDataContext) TargetAt(i int) string      { return "" }
func (m *mockDataContext) AddTarget(target string)    {}
func (m *mockDataContext) Dropped() bool              { return false }
func (m *mockDataContext) SetDropped(v bool)          {}
func (m *mockDataContext) Timestamp() int64           { return 1000 }
func (m *mockDataContext) SpanContext() contract.SpanContext { return contract.SpanContext{} }
func (m *mockDataContext) SetSpanContext(sc contract.SpanContext) {}
func (m *mockDataContext) PreviousValue() (float64, bool) { return m.prevValue, m.hasPrevValue }
func (m *mockDataContext) SetPreviousValue(v float64) { m.prevValue = v; m.hasPrevValue = true }
func (m *mockDataContext) Raw() any                   { return nil }

// ─────────────────────────────────────────────
//  BitMaskCondition 测试
// ─────────────────────────────────────────────

func TestBitMaskCondition(t *testing.T) {
	tests := []struct {
		name     string
		mask     uint64
		expected uint64
		operator string
		value    float64
		want     bool
	}{
		{"and_match", 0x0F, 0x05, "and", 0x15, true},  // 0x15 & 0x0F = 0x05
		{"and_no_match", 0x0F, 0x05, "and", 0x16, false}, // 0x16 & 0x0F = 0x06
		{"or_match", 0xF0, 0xF5, "or", 0x05, true},   // 0x05 | 0xF0 = 0xF5
		{"eq_match", 0, 0x10, "eq", 0x10, true},
		{"eq_no_match", 0, 0x10, "eq", 0x11, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := condition.NewBitMaskCondition("test", tt.mask, tt.expected, tt.operator)
			data := newMockData()
			data.value = tt.value

			got := cond.Evaluate(context.Background(), data)
			if got != tt.want {
				t.Errorf("BitMaskCondition.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
//  DeltaThresholdCondition 测试
// ─────────────────────────────────────────────

func TestDeltaThresholdCondition(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		direction string
		prevValue float64
		currValue float64
		want      bool
	}{
		{"up_match", 10.0, "up", 50.0, 65.0, true},
		{"up_no_match", 10.0, "up", 50.0, 55.0, false},
		{"down_match", 10.0, "down", 50.0, 35.0, true},
		{"down_no_match", 10.0, "down", 50.0, 45.0, false},
		{"both_up", 10.0, "both", 50.0, 65.0, true},
		{"both_down", 10.0, "both", 50.0, 35.0, true},
		{"both_no_match", 10.0, "both", 50.0, 55.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := condition.NewDeltaThresholdCondition("test", tt.threshold, tt.direction)
			data := newMockData()
			data.value = tt.currValue
			data.prevValue = tt.prevValue
			data.hasPrevValue = true

			got := cond.Evaluate(context.Background(), data)
			if got != tt.want {
				t.Errorf("DeltaThresholdCondition.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
//  RateLimitCondition 测试
// ─────────────────────────────────────────────

func TestRateLimitCondition(t *testing.T) {
	tests := []struct {
		name          string
		rateThreshold float64
		direction     string
		prevValue     float64
		currValue     float64
		prevTs        int64
		currTs        int64
		want          bool
	}{
		{"up_match", 10.0, "up", 50.0, 70.0, 0, 1000, true}, // rate = 20/s
		{"up_no_match", 10.0, "up", 50.0, 55.0, 0, 1000, false}, // rate = 5/s
		{"down_match", 10.0, "down", 50.0, 30.0, 0, 1000, true}, // rate = -20/s
		{"both_match", 10.0, "both", 50.0, 70.0, 0, 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := condition.NewRateLimitCondition("test", tt.rateThreshold, tt.direction)
			data := newMockData()
			data.value = tt.currValue
			data.prevValue = tt.prevValue
			data.hasPrevValue = true
			data.tags["_prev_timestamp"] = "0"

			got := cond.Evaluate(context.Background(), data)
			if got != tt.want {
				t.Errorf("RateLimitCondition.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}