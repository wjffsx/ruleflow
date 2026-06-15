package util

import "testing"

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
		ok       bool
	}{
		{"float64", 10.5, 10.5, true},
		{"float32", float32(10.5), 10.5, true},
		{"int", 10, 10.0, true},
		{"int64", int64(10), 10.0, true},
		{"int32", int32(10), 10.0, true},
		{"string", "10", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := ToFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("expected ok=%v, got ok=%v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStateKey(t *testing.T) {
	key := StateKey("duration", "cond_001", "device001", "power")
	expected := "duration:cond_001:device001/power"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestStateKeySimple(t *testing.T) {
	key := StateKeySimple("trend", "device001", "frequency")
	expected := "trend:device001/frequency"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}