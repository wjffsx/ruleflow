// Package profiler 提供节点级性能分析器。
//
// 应用层需要 Top N 时，从 contrib/memorysink 聚合，
// 或使用本 Profiler 作为独立的聚合工具。
package profiler

import (
	"sort"
	"sync"
)

// NodeProfile 节点性能概要
type NodeProfile struct {
	ChainID      string  `json:"chain_id"`
	NodeID       string  `json:"node_id"`
	NodeType     string  `json:"node_type"` // "condition" | "action" | "action:{type}"
	ExecCount    int64   `json:"exec_count"`
	AvgLatencyNs float64 `json:"avg_latency_ns"`
	MaxLatencyNs int64   `json:"max_latency_ns"`
	ErrorCount   int64   `json:"error_count"`
}

// profileKey 生成 map 中的唯一键
func profileKey(chainID, nodeID, nodeType string) string {
	return chainID + "|" + nodeID + "|" + nodeType
}

// Profiler 节点性能分析器，收集并排序节点级执行耗时。
// Profiler 是 goroutine-safe 的。
type Profiler struct {
	mu    sync.Mutex
	nodes map[string]*NodeProfile // profileKey -> profile
}

// NewProfiler 创建 Profiler
func NewProfiler() *Profiler {
	return &Profiler{nodes: make(map[string]*NodeProfile)}
}

// Record 记录一次节点执行
func (p *Profiler) Record(chainID, nodeID, nodeType string, latencyNs int64, hasError bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := profileKey(chainID, nodeID, nodeType)
	profile, ok := p.nodes[key]
	if !ok {
		profile = &NodeProfile{
			ChainID:  chainID,
			NodeID:   nodeID,
			NodeType: nodeType,
		}
		p.nodes[key] = profile
	}

	profile.ExecCount++
	profile.AvgLatencyNs = profile.AvgLatencyNs + (float64(latencyNs)-profile.AvgLatencyNs)/float64(profile.ExecCount)
	if latencyNs > profile.MaxLatencyNs {
		profile.MaxLatencyNs = latencyNs
	}
	if hasError {
		profile.ErrorCount++
	}
}

// TopByMaxLatency 按最大耗时降序返回 Top N
func (p *Profiler) TopByMaxLatency(n int) []NodeProfile {
	p.mu.Lock()
	profiles := make([]NodeProfile, 0, len(p.nodes))
	for _, v := range p.nodes {
		profiles = append(profiles, *v)
	}
	p.mu.Unlock()

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].MaxLatencyNs > profiles[j].MaxLatencyNs
	})
	if n <= 0 || n >= len(profiles) {
		return profiles
	}
	return profiles[:n]
}

// TopByAvgLatency 按平均耗时降序返回 Top N
func (p *Profiler) TopByAvgLatency(n int) []NodeProfile {
	p.mu.Lock()
	profiles := make([]NodeProfile, 0, len(p.nodes))
	for _, v := range p.nodes {
		profiles = append(profiles, *v)
	}
	p.mu.Unlock()

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].AvgLatencyNs > profiles[j].AvgLatencyNs
	})
	if n <= 0 || n >= len(profiles) {
		return profiles
	}
	return profiles[:n]
}

// TopByExecCount 按执行次数降序返回 Top N
func (p *Profiler) TopByExecCount(n int) []NodeProfile {
	p.mu.Lock()
	profiles := make([]NodeProfile, 0, len(p.nodes))
	for _, v := range p.nodes {
		profiles = append(profiles, *v)
	}
	p.mu.Unlock()

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].ExecCount > profiles[j].ExecCount
	})
	if n <= 0 || n >= len(profiles) {
		return profiles
	}
	return profiles[:n]
}

// Snapshot 返回所有 profile 的快照
func (p *Profiler) Snapshot() []NodeProfile {
	p.mu.Lock()
	defer p.mu.Unlock()

	profiles := make([]NodeProfile, 0, len(p.nodes))
	for _, v := range p.nodes {
		profiles = append(profiles, *v)
	}
	return profiles
}

// Reset 清空所有 profile
func (p *Profiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nodes = make(map[string]*NodeProfile)
}
