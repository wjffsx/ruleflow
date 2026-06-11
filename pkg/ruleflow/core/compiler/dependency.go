package compiler

import (
	"fmt"
	"strings"
	"sync"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DependencyGraph — 规则链依赖图
// ─────────────────────────────────────────────

// DependencyGraph 规则链依赖图。
// 记录链与链之间的引用关系，支持：
//   - 添加 / 移除节点
//   - 循环引用检测（DFS 三色标记）
//   - 拓扑排序
//   - 反向引用查询（被哪些链引用）
//
// 线程安全：所有方法使用读写锁保护。
type DependencyGraph struct {
	mu sync.RWMutex
	// nodes 记录所有节点（key 为 chainID）
	nodes map[string]struct{}
	// edges 正向边：A -> B 表示 A 依赖 B
	edges map[string][]string
	// reverseEdges 反向边：B -> []A 表示 B 被哪些链依赖
	reverseEdges map[string][]string
}

// NewDependencyGraph 创建依赖图
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:        make(map[string]struct{}),
		edges:        make(map[string][]string),
		reverseEdges: make(map[string][]string),
	}
}

// AddNode 添加节点
func (g *DependencyGraph) AddNode(chainID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.nodes[chainID]; !ok {
		g.nodes[chainID] = struct{}{}
		g.edges[chainID] = nil
	}
}

// AddEdge 添加依赖边：from 依赖 to。
// 如果添加后形成循环，返回 error 且不修改图。
func (g *DependencyGraph) AddEdge(from, to string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 自引用检测
	if from == to {
		return core.NewDependencyError(from, []string{from, from})
	}

	// 确保节点存在
	if _, ok := g.nodes[from]; !ok {
		g.nodes[from] = struct{}{}
		g.edges[from] = nil
	}
	if _, ok := g.nodes[to]; !ok {
		g.nodes[to] = struct{}{}
		g.edges[to] = nil
	}

	// 临时添加边并检测循环
	originalEdges := g.edges[from]
	g.edges[from] = append(g.edges[from], to)
	if cycle, hasCycle := g.detectCycleLocked(from); hasCycle {
		// 回滚
		g.edges[from] = originalEdges
		return core.NewDependencyError(from, cycle)
	}

	// 记录反向边
	g.reverseEdges[to] = append(g.reverseEdges[to], from)
	return nil
}

// RemoveNode 移除节点及其所有边
func (g *DependencyGraph) RemoveNode(chainID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[chainID]; !ok {
		return
	}

	// 清理反向边
	for _, dep := range g.edges[chainID] {
		g.reverseEdges[dep] = removeFromSlice(g.reverseEdges[dep], chainID)
	}

	// 清理其他节点指向此节点的反向边
	for _, referrer := range g.reverseEdges[chainID] {
		g.edges[referrer] = removeFromSlice(g.edges[referrer], chainID)
	}

	delete(g.nodes, chainID)
	delete(g.edges, chainID)
	delete(g.reverseEdges, chainID)
}

// DetectCycle 从所有节点出发检测循环
func (g *DependencyGraph) DetectCycle() ([]string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for node := range g.nodes {
		if cycle, ok := g.detectCycleLocked(node); ok {
			return cycle, true
		}
	}
	return nil, false
}

// GetDependencies 返回指定链依赖的其他链 ID 列表
func (g *DependencyGraph) GetDependencies(chainID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	deps := g.edges[chainID]
	if len(deps) == 0 {
		return nil
	}
	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

// GetReferencedBy 返回引用了指定链的其他链 ID 列表
func (g *DependencyGraph) GetReferencedBy(chainID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	refs := g.reverseEdges[chainID]
	if len(refs) == 0 {
		return nil
	}
	result := make([]string, len(refs))
	copy(result, refs)
	return result
}

// HasNode 检查节点是否存在
func (g *DependencyGraph) HasNode(chainID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.nodes[chainID]
	return ok
}

// NodeCount 返回节点数量
func (g *DependencyGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// TopoSort 拓扑排序，返回排序后的链 ID 列表。
// 如果存在循环，返回 error。
func (g *DependencyGraph) TopoSort() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 计算每个节点的入度（被依赖数量）
	inDegree := make(map[string]int, len(g.nodes))
	for node := range g.nodes {
		inDegree[node] = 0
	}
	for _, deps := range g.edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// 入度为 0 的节点进入队列
	queue := make([]string, 0, len(g.nodes))
	for node, d := range inDegree {
		if d == 0 {
			queue = append(queue, node)
		}
	}

	result := make([]string, 0, len(g.nodes))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)
		for _, dep := range g.edges[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("%w: topo sort failed, %d nodes unreachable", core.ErrCyclicDependency, len(g.nodes)-len(result))
	}
	return result, nil
}

// String 返回图的可读表示
func (g *DependencyGraph) String() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DependencyGraph(%d nodes):\n", len(g.nodes)))
	for node, deps := range g.edges {
		if len(deps) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %s -> %v\n", node, deps))
	}
	return sb.String()
}

// ─────────────────────────────────────────────
//  内部方法
// ─────────────────────────────────────────────

// detectCycleLocked 从指定起点检测循环（DFS 三色标记）。
// 调用方必须持有锁。
func (g *DependencyGraph) detectCycleLocked(start string) ([]string, bool) {
	const (
		white = 0 // 未访问
		gray  = 1 // 正在访问（在当前 DFS 路径上）
		black = 2 // 访问完成
	)

	color := make(map[string]int, len(g.nodes))
	parent := make(map[string]string, len(g.nodes))

	var cyclePath []string
	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, next := range g.edges[node] {
			switch color[next] {
			case gray:
				// 找到循环，回溯路径
				cyclePath = []string{next, node}
				cur := node
				for cur != next && parent[cur] != "" {
					cur = parent[cur]
					cyclePath = append(cyclePath, cur)
				}
				return true
			case white:
				parent[next] = node
				if dfs(next) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	if dfs(start) {
		return cyclePath, true
	}
	return nil, false
}

func removeFromSlice(s []string, target string) []string {
	for i, v := range s {
		if v == target {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
