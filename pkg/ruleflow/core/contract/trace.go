// Package contract - 追踪上下文类型
package contract

// TraceID 16 字节 trace 标识
type TraceID [16]byte

// SpanID 8 字节 span 标识
type SpanID [8]byte

// SpanContext 表示一个 span 的最小追踪上下文
type SpanContext struct {
	TraceID    TraceID
	SpanID     SpanID
	TraceFlags byte
	TraceState TraceState
}

// TraceState W3C trace state 的最小实现
type TraceState struct {
	List []TraceStateEntry
}

// TraceStateEntry 单条 trace state 键值对
type TraceStateEntry struct {
	Key   string
	Value string
}

// IsZero 判断 SpanContext 是否为零值
func (sc SpanContext) IsZero() bool {
	return sc.TraceID == [16]byte{} && sc.SpanID == [8]byte{}
}

// IsSampled 判断 span 是否被采样
func (sc SpanContext) IsSampled() bool {
	return sc.TraceFlags&0x01 == 0x01
}

// WithSampledFlag 返回带新采样标志的 SpanContext
func (sc SpanContext) WithSampledFlag(sampled bool) SpanContext {
	flags := sc.TraceFlags
	if sampled {
		flags |= 0x01
	} else {
		flags &^= 0x01
	}
	sc.TraceFlags = flags
	return sc
}

// IsValid 判断是否为合法 SpanContext
func (sc SpanContext) IsValid() bool {
	return sc.TraceID != [16]byte{} && sc.SpanID != [8]byte{}
}
