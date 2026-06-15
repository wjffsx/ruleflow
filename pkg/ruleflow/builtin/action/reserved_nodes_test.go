// Package action_test provides tests for builtin action nodes
package action_test

import (
	"context"
	"testing"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/action"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// mockDataContext 用于测试的模拟 DataContext
type mockDataContext struct {
	deviceID  string
	pointName string
	value     float64
	quality   int
	tags      map[string]string
}

func newMockData() *mockDataContext {
	return &mockDataContext{
		deviceID:  "device001",
		pointName: "status",
		tags:      make(map[string]string),
	}
}

func (m *mockDataContext) DeviceID() string                       { return m.deviceID }
func (m *mockDataContext) PointName() string                      { return m.pointName }
func (m *mockDataContext) SetPointName(name string)               { m.pointName = name }
func (m *mockDataContext) PointType() string                      { return "digital" }
func (m *mockDataContext) FQN() string                            { return m.deviceID + "/" + m.pointName }
func (m *mockDataContext) Value() float64                         { return m.value }
func (m *mockDataContext) SetValue(v float64)                     { m.value = v }
func (m *mockDataContext) Quality() int                           { return m.quality }
func (m *mockDataContext) SetQuality(q int)                       { m.quality = q }
func (m *mockDataContext) UpperLimit() (float64, bool)            { return 0, false }
func (m *mockDataContext) LowerLimit() (float64, bool)            { return 0, false }
func (m *mockDataContext) LimitExceeded() bool                    { return false }
func (m *mockDataContext) SetLimitExceeded(v bool)                {}
func (m *mockDataContext) GetTag(key string) string               { return m.tags[key] }
func (m *mockDataContext) SetTag(key, value string)               { m.tags[key] = value }
func (m *mockDataContext) TargetCount() int                       { return 0 }
func (m *mockDataContext) TargetAt(i int) string                  { return "" }
func (m *mockDataContext) AddTarget(target string)                {}
func (m *mockDataContext) Dropped() bool                          { return false }
func (m *mockDataContext) SetDropped(v bool)                      {}
func (m *mockDataContext) Timestamp() int64                       { return 1000 }
func (m *mockDataContext) SpanContext() contract.SpanContext      { return contract.SpanContext{} }
func (m *mockDataContext) SetSpanContext(sc contract.SpanContext) {}
func (m *mockDataContext) PreviousValue() (float64, bool)         { return 0, false }
func (m *mockDataContext) SetPreviousValue(v float64)             {}
func (m *mockDataContext) Raw() any                               { return nil }

// ─────────────────────────────────────────────
//  BitUnpackAction 测试
// ─────────────────────────────────────────────

func TestBitUnpackAction(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		outputTags []string
		startBit   int
		wantTags   map[string]string
	}{
		{
			name:       "unpack_3bits",
			value:      0b101, // 5
			outputTags: []string{"bit0", "bit1", "bit2"},
			startBit:   0,
			wantTags:   map[string]string{"bit0": "1", "bit1": "0", "bit2": "1"},
		},
		{
			name:       "unpack_with_offset",
			value:      0b1100, // 12
			outputTags: []string{"bit2", "bit3"},
			startBit:   2,
			wantTags:   map[string]string{"bit2": "1", "bit3": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := action.NewBitUnpackAction("test", tt.outputTags, tt.startBit)
			data := newMockData()
			data.value = tt.value

			err := act.Execute(context.Background(), data)
			if err != nil {
				t.Errorf("BitUnpackAction.Execute() error = %v", err)
			}

			for key, want := range tt.wantTags {
				got := data.GetTag(key)
				if got != want {
					t.Errorf("tag[%s] = %v, want %v", key, got, want)
				}
			}
		})
	}
}

// ─────────────────────────────────────────────
//  BitPackAction 测试
// ─────────────────────────────────────────────

func TestBitPackAction(t *testing.T) {
	tests := []struct {
		name        string
		inputTags   map[string]string
		outputField string
		startBit    int
		wantValue   float64
		wantTag     string
	}{
		{
			name:        "pack_3bits_to_value",
			inputTags:   map[string]string{"bit0": "1", "bit1": "0", "bit2": "1"},
			outputField: "value",
			startBit:    0,
			wantValue:   5, // 0b101
		},
		{
			name:        "pack_with_offset",
			inputTags:   map[string]string{"bit0": "1", "bit1": "1"},
			outputField: "result",
			startBit:    2,
			wantTag:     "12", // 0b1100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := action.NewBitPackAction("test", []string{"bit0", "bit1", "bit2"}, tt.outputField, tt.startBit)
			data := newMockData()

			// 设置输入 tags
			for key, val := range tt.inputTags {
				data.SetTag(key, val)
			}

			err := act.Execute(context.Background(), data)
			if err != nil {
				t.Errorf("BitPackAction.Execute() error = %v", err)
			}

			if tt.outputField == "value" {
				if data.Value() != tt.wantValue {
					t.Errorf("value = %v, want %v", data.Value(), tt.wantValue)
				}
			} else {
				got := data.GetTag(tt.outputField)
				if got != tt.wantTag {
					t.Errorf("tag[%s] = %v, want %v", tt.outputField, got, tt.wantTag)
				}
			}
		})
	}
}
