package slog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSlogLogger_OutputsStructuredLog(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	l := NewSlogLogger(inner)

	l.Info("test message", "key1", "v1", "key2", 42)

	out := buf.String()
	if !strings.Contains(out, `"msg":"test message"`) {
		t.Errorf("missing msg in: %s", out)
	}
	if !strings.Contains(out, `"key1":"v1"`) {
		t.Errorf("missing key1 in: %s", out)
	}
	if !strings.Contains(out, `"key2":42`) {
		t.Errorf("missing key2 in: %s", out)
	}
}

func TestSlogLogger_RespectsLogLevel(t *testing.T) {
	var buf bytes.Buffer
	// 设为 Warn 级别
	inner := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	l := NewSlogLogger(inner)

	l.Debug("debug-msg")
	l.Info("info-msg")
	l.Warn("warn-msg")
	l.Error("error-msg")

	out := buf.String()
	if strings.Contains(out, "debug-msg") {
		t.Errorf("debug should be filtered: %s", out)
	}
	if strings.Contains(out, "info-msg") {
		t.Errorf("info should be filtered: %s", out)
	}
	if !strings.Contains(out, "warn-msg") {
		t.Errorf("warn should be present: %s", out)
	}
	if !strings.Contains(out, "error-msg") {
		t.Errorf("error should be present: %s", out)
	}
}

func TestSlogLogger_With_AttachesFields(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	l := NewSlogLogger(inner)

	child := l.With("component", "engine")
	child.Info("started")

	out := buf.String()
	if !strings.Contains(out, `"component":"engine"`) {
		t.Errorf("missing component field: %s", out)
	}
}

func TestSlogLogger_NilLogger_UsesDefault(t *testing.T) {
	l := NewSlogLogger(nil)
	if l == nil || l.logger == nil {
		t.Fatal("NewSlogLogger(nil) should return a non-nil logger using default")
	}
	// 调用方法不 panic
	l.Info("ok")
}

func TestSlogFromLogger_BridgeSlogToCustom(t *testing.T) {
	var buf bytes.Buffer
	custom := NewSlogLogger(slog.New(slog.NewJSONHandler(&buf, nil)))
	slogLogger := SlogFromLogger(custom)
	slogLogger.Info("hello", "k", "v")
	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Errorf("slog bridge failed: %s", out)
	}
	if !strings.Contains(out, `"k":"v"`) {
		t.Errorf("slog bridge missing field: %s", out)
	}
}
