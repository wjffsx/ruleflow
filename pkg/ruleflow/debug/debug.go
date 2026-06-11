package debug

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// DebugMode 调试模式
type DebugMode int

const (
	DebugOff      DebugMode = iota // 关闭调试（默认）
	DebugAll                       // 记录所有输入和输出事件
	DebugFailures                  // 仅记录 Failure/Error 事件
)

// DebugMode 的字符串表示
func (m DebugMode) String() string {
	switch m {
	case DebugAll:
		return "all"
	case DebugFailures:
		return "failures"
	default:
		return "off"
	}
}

// EventType 调试事件类型
type EventType string

const (
	EventIn  EventType = "in"  // 节点输入事件
	EventOut EventType = "out" // 节点输出事件
)

// DebugEvent 调试事件
type DebugEvent struct {
	EventType    EventType `json:"event_type"`
	ChainID      string    `json:"chain_id"`
	RuleID       string    `json:"rule_id"`
	NodeID       string    `json:"node_id"`
	NodeType     string    `json:"node_type"`               // "condition" | "action:{type}"
	RelationType string    `json:"relation_type"`           // 连接类型："matched"|"unmatched"|"error"
	DataSnapshot string    `json:"data_snapshot,omitempty"` // 数据快照 JSON
	Error        string    `json:"error,omitempty"`
	Timestamp    int64     `json:"timestamp"`
	DurationNs   int64     `json:"duration_ns,omitempty"`
}

// DebugSink 调试事件输出接口（可插拔）
// V12：参数类型改为 any，兼容 contract.DebugSink
type DebugSink interface {
	// WriteEvent 写入一条调试事件
	WriteEvent(ctx any, event DebugEvent) error
}

// DebugManager 调试管理器
// V12：实现 contract.DebugManager 接口，供 engine 通过接口调用
type DebugManager struct {
	mode     atomic.Int64 // 存储 DebugMode 的 int64 值
	sink     DebugSink
	deadline atomic.Int64 // DebugAll 模式截止时间戳（纳秒），0 表示不限制
	// V4.9：perSecLimit 限流字段已迁出至 contrib/debug/ratelimit_sink（QoS 越界收敛）
}

// 确保 DebugManager 实现 contract.DebugManager 接口
var _ contract.DebugManager = (*DebugManager)(nil)

// NewDebugManager 创建调试管理器（V4.9 收敛：perSecLimit 参数已删除）
//   - mode: 调试模式
//   - sink: 事件输出（nil 时为 noop）
//   - allDeadline: DebugAll 模式截止时间（零值表示不限制）
//
// 限流（每秒最大事件数）已迁出至 contrib/debug/ratelimit_sink 包装层：
//
//	bus := debug.NewChannelSink(1024)
//	limitedSink := debug.NewRateLimitSink(bus, 100)
//	mgr := coredebug.NewDebugManager(coredebug.DebugAll, limitedSink, time.Time{})
func NewDebugManager(mode DebugMode, sink DebugSink, allDeadline time.Time) *DebugManager {
	dm := &DebugManager{sink: sink}
	dm.mode.Store(int64(mode))
	if !allDeadline.IsZero() {
		dm.deadline.Store(allDeadline.UnixNano())
	}
	return dm
}

// Mode 返回当前调试模式（纯 getter，不修改状态）
// 注意：deadline 过期检测在 Capture 中处理。
func (dm *DebugManager) Mode() DebugMode {
	return DebugMode(dm.mode.Load())
}

// SetMode 动态设置调试模式
func (dm *DebugManager) SetMode(mode DebugMode, allDeadline time.Time) {
	dm.mode.Store(int64(mode))
	if mode == DebugAll && !allDeadline.IsZero() {
		dm.deadline.Store(allDeadline.UnixNano())
	} else {
		dm.deadline.Store(0)
	}
	// V4.8：不再写 enabled 字段；Enabled() 方法自动从 mode + sink 推导
}

// SetSink 设置或更换输出实现
func (dm *DebugManager) SetSink(sink DebugSink) {
	dm.sink = sink
	// V4.8：不再写 enabled 字段；Enabled() 方法自动从 mode + sink 推导
}

// Enabled 返回调试是否启用（V4.8：方法，取代 enabled 字段）
//
// 由 (mode != DebugOff && sink != nil) 推导，避免与 mode/sink 双状态不一致。
// 性能：1 次 atomic load + 1 次 nil 比较 + 1 次 enum 等值，编译器可内联。
func (dm *DebugManager) Enabled() bool {
	return DebugMode(dm.mode.Load()) != DebugOff && dm.sink != nil
}

// ShouldCapture 判断是否应该捕获当前事件（不触发副作用）
//   - relationType: "matched" | "unmatched" | "error" | "dropped"
//
// 注意：该方法不进行速率限制检查，也不处理 deadline 过期；
// 仅做模式过滤。速率限制和 deadline 过期由 Capture 统一处理。
func (dm *DebugManager) ShouldCapture(relationType string) bool {
	if !dm.Enabled() {
		return false
	}
	mode := dm.Mode()
	if mode == DebugOff {
		return false
	}
	if mode == DebugFailures {
		return relationType == "error" || relationType == "dropped"
	}
	// DebugAll 模式：捕获所有（速率限制由 Capture 处理）
	return true
}

// Capture 捕获一条调试事件（统一处理所有过滤逻辑）
// V12：实现 contract.DebugManager 接口
//
// 处理顺序（按短路优化）：
//  1. enabled 快速路径
//  2. 模式读取（DebugOff 直接返回）
//  3. DebugAll 模式的 deadline 过期检查 + 自动关闭
//  4. 模式过滤（DebugFailures 仅记录 error/dropped）
//  5. 写入 sink（V4.9：限流下沉至 contrib/debug/ratelimit_sink 包装层）
func (dm *DebugManager) Capture(ctx any, event contract.DebugEvent) {
	if !dm.Enabled() {
		return
	}
	mode := dm.Mode()
	if mode == DebugOff {
		return
	}
	// DebugAll 模式：检查 deadline 是否过期
	if mode == DebugAll && dm.isDeadlineExpired() {
		dm.disable()
		return
	}
	// 模式过滤
	if mode == DebugFailures && event.RelationType != "error" && event.RelationType != "dropped" {
		return
	}

	event.Timestamp = time.Now().UnixNano()
	if dm.sink != nil {
		// 转换为内部 DebugEvent 类型
		internalEvent := DebugEvent{
			EventType:    EventType(event.EventType),
			ChainID:      event.ChainID,
			RuleID:       event.RuleID,
			NodeID:       event.NodeID,
			NodeType:     event.NodeType,
			RelationType: event.RelationType,
			DataSnapshot: event.DataSnapshot,
			Error:        event.Error,
			Timestamp:    event.Timestamp,
			DurationNs:   event.DurationNs,
		}
		_ = dm.sink.WriteEvent(ctx, internalEvent)
	}
}

// CaptureIn 便捷方法：捕获节点输入事件
// V12：实现 contract.DebugManager 接口
func (dm *DebugManager) CaptureIn(ctx any, chainID, ruleID, nodeID, nodeType, dataSnapshot string) {
	dm.Capture(ctx, contract.DebugEvent{
		EventType:    contract.DebugEventIn,
		ChainID:      chainID,
		RuleID:       ruleID,
		NodeID:       nodeID,
		NodeType:     nodeType,
		RelationType: "in",
		DataSnapshot: dataSnapshot,
	})
}

// CaptureOut 便捷方法：捕获节点输出事件
// V12：实现 contract.DebugManager 接口
func (dm *DebugManager) CaptureOut(ctx any, chainID, ruleID, nodeID, nodeType, relationType, dataSnapshot string, durationNs int64, errMsg string) {
	dm.Capture(ctx, contract.DebugEvent{
		EventType:    contract.DebugEventOut,
		ChainID:      chainID,
		RuleID:       ruleID,
		NodeID:       nodeID,
		NodeType:     nodeType,
		RelationType: relationType,
		DataSnapshot: dataSnapshot,
		DurationNs:   durationNs,
		Error:        errMsg,
	})
}

// isDeadlineExpired 检查 DebugAll 模式的 deadline 是否过期
func (dm *DebugManager) isDeadlineExpired() bool {
	d := dm.deadline.Load()
	if d <= 0 {
		return false
	}
	return time.Now().UnixNano() > d
}

// disable 关闭调试（单一入口：避免在多个地方重复设置状态）
// V4.8：仅需将 mode 置为 DebugOff；Enabled() 方法自动返回 false（无需写 enabled 字段）
func (dm *DebugManager) disable() {
	dm.mode.Store(int64(DebugOff))
}

// V4.9：checkRateLimit 已删除（限流下沉至 contrib/debug/ratelimit_sink 包装层）

// ─────────────────────────────────────────────
//  noopSink — 空实现
// ─────────────────────────────────────────────

type noopSink struct{}

func (noopSink) WriteEvent(_ any, _ DebugEvent) error { return nil }

// NoopSink 返回空事件输出（默认行为）
func NoopSink() DebugSink { return noopSink{} }
