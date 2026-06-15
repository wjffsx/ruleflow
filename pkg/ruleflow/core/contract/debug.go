// Package contract defines the contract layer for the ruleflow engine.
//
// This package contains pure interfaces, enumerations, value types, and
// zero-logic noop default implementations.
package contract

// ─────────────────────────────────────────────
//  Debug — 调试事件接口
// ─────────────────────────────────────────────

// DebugEventType 调试事件类型
type DebugEventType string

const (
	DebugEventIn  DebugEventType = "in"  // 节点输入事件
	DebugEventOut DebugEventType = "out" // 节点输出事件
)

// DebugEvent 调试事件结构
type DebugEvent struct {
	EventType    DebugEventType `json:"event_type"`
	ChainID      string         `json:"chain_id"`
	RuleID       string         `json:"rule_id"`
	NodeID       string         `json:"node_id"`
	NodeType     string         `json:"node_type"`               // "condition" | "action:{type}"
	RelationType string         `json:"relation_type"`           // "matched"|"unmatched"|"error"
	DataSnapshot string         `json:"data_snapshot,omitempty"` // 数据快照 JSON
	Error        string         `json:"error,omitempty"`
	Timestamp    int64          `json:"timestamp"`
	DurationNs   int64          `json:"duration_ns,omitempty"`
}

// DebugSink 调试事件输出接口
type DebugSink interface {
	WriteEvent(ctx any, event DebugEvent) error
}

// DebugManager 调试管理器接口
// 核心库只依赖此接口，具体实现由 debug 包提供
type DebugManager interface {
	// Enabled 返回调试是否启用
	Enabled() bool
	// ShouldCapture 判断是否应该捕获当前事件
	ShouldCapture(relationType string) bool
	// Capture 捕获一条调试事件
	Capture(ctx any, event DebugEvent)
	// CaptureIn 捕获节点输入事件
	CaptureIn(ctx any, chainID, ruleID, nodeID, nodeType, dataSnapshot string)
	// CaptureOut 捕获节点输出事件
	CaptureOut(ctx any, chainID, ruleID, nodeID, nodeType, relationType, dataSnapshot string, durationNs int64, errMsg string)
}

// NoopDebugManager 空调试管理器实现
type NoopDebugManager struct{}

func (NoopDebugManager) Enabled() bool                                                { return false }
func (NoopDebugManager) ShouldCapture(_ string) bool                                  { return false }
func (NoopDebugManager) Capture(_ any, _ DebugEvent)                                  {}
func (NoopDebugManager) CaptureIn(_ any, _, _, _, _, _ string)                        {}
func (NoopDebugManager) CaptureOut(_ any, _, _, _, _, _, _ string, _ int64, _ string) {}

// 编译期接口检查
var _ DebugManager = NoopDebugManager{}
