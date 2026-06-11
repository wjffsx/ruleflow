// Package debuginternal 提供 observability-grpc-debug 示例专用的 DebugService 工具。
//
// 本包是 examples/observability-grpc-debug 的内部子包：仅作示例代码使用，不对外暴露。
// SubscribeFilter 实现的是"协议层"职责——在 EventBus → gRPC stream 之间
// 按订阅条件过滤事件，避免向客户端推送不需要的事件。
//
// 设计背景：V1 的 contrib/debug/filter.go 已删除（属于协议层职责）；
// 本包是它的迁移目标，作为 DebugService 的辅助工具存在。
package debuginternal

import (
	coredebug "github.com/wjffsx/ruleflow/pkg/ruleflow/debug"
)

// SubscribeFilter 服务端过滤条件
// 在 EventBus → gRPC stream 之间进行过滤，减少网络传输。
type SubscribeFilter struct {
	chainIDs   map[string]struct{}
	ruleIDs    map[string]struct{}
	nodeType   string
	onlyErrors bool
}

// NewSubscribeFilter 创建过滤器
func NewSubscribeFilter(chainIDs, ruleIDs []string, nodeType string, onlyErrors bool) *SubscribeFilter {
	f := &SubscribeFilter{
		nodeType:   nodeType,
		onlyErrors: onlyErrors,
	}
	if len(chainIDs) > 0 {
		f.chainIDs = make(map[string]struct{}, len(chainIDs))
		for _, id := range chainIDs {
			f.chainIDs[id] = struct{}{}
		}
	}
	if len(ruleIDs) > 0 {
		f.ruleIDs = make(map[string]struct{}, len(ruleIDs))
		for _, id := range ruleIDs {
			f.ruleIDs[id] = struct{}{}
		}
	}
	return f
}

// Match 判断事件是否匹配过滤条件
func (f *SubscribeFilter) Match(event coredebug.DebugEvent) bool {
	if f.onlyErrors && event.RelationType != "error" && event.RelationType != "dropped" {
		return false
	}
	if f.chainIDs != nil {
		if _, ok := f.chainIDs[event.ChainID]; !ok {
			return false
		}
	}
	if f.ruleIDs != nil {
		if _, ok := f.ruleIDs[event.RuleID]; !ok {
			return false
		}
	}
	if f.nodeType != "" && event.NodeType != f.nodeType {
		return false
	}
	return true
}
