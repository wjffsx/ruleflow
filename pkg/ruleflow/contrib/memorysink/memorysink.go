// Package memorysink 提供 ruleflow 引擎的内存版 MetricsSink 实现。
//
// 用途：
//   - 应用层希望快速接入但不想引入 Prometheus 等外部依赖的场景
//   - 单元测试断言指标
//
// 设计目标：
//   - 线程安全：所有方法可在并发引擎热路径上调用
//   - Snapshot 一致性：Snapshot() 返回深拷贝，可安全用于并发读
//
// 基本用法：
//
//	import "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/memorysink"
//
//	sink := memorysink.NewMemorySink()
//	engine := core.NewEngine(core.WithMetricsSink(sink))
//
//	// 周期性获取快照
//	go func() {
//	    for {
//	        snap := sink.Snapshot()
//	        log.Printf("eval: %+v", snap.EvalTotal)
//	        time.Sleep(5 * time.Second)
//	    }
//	}()
package memorysink

import (
	"sync"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// Snapshot MemorySink 的线程安全快照
type Snapshot struct {
	EvalTotal     map[string]map[string]int64
	ActionTotal   map[string]map[string]map[string]int64
	RuleThrottled map[string]map[string]int64
	Panic         map[string]map[string]int64
	EvalDuration  map[string]int64
	EvalCount     map[string]int64
	ConditionEval map[string]map[string]map[string]int64
	ActionExec    map[string]map[string]map[string]int64
	LoadedChains  int
	ActiveEval    int64
}

// MemorySink 简单的内存计数器实现，可用于：
//   - 应用层希望快速接入但不想引入 Prometheus 等外部依赖的场景
//   - 单元测试断言指标
type MemorySink struct {
	mu sync.Mutex
	// EvalTotal 按 (chainID, result) 计数
	EvalTotal map[string]map[string]int64
	// ActionTotal 按 (chainID, actionType, result) 计数
	ActionTotal map[string]map[string]map[string]int64
	// RuleThrottled 按 (chainID, ruleID) 计数
	RuleThrottled map[string]map[string]int64
	// Panic 按 (chainID, ruleID) 计数
	Panic map[string]map[string]int64
	// EvalDuration 按 chainID 累计（纳秒）
	EvalDuration map[string]int64
	// EvalCount 按 chainID 计数
	EvalCount map[string]int64
	// ConditionEval 按 (chainID, nodeID) 记录条件评估 metrics: count, matched, latency_ns
	ConditionEval map[string]map[string]map[string]int64
	// ActionExec 按 (chainID, nodeID) 记录动作执行 metrics: count, error, latency_ns
	ActionExec map[string]map[string]map[string]int64
	// LoadedChains 已加载链数
	LoadedChains int
	// ActiveEval 当前活跃评估数
	ActiveEval int64
}

// NewMemorySink 创建 MemorySink
func NewMemorySink() *MemorySink {
	return &MemorySink{
		EvalTotal:     make(map[string]map[string]int64),
		ActionTotal:   make(map[string]map[string]map[string]int64),
		RuleThrottled: make(map[string]map[string]int64),
		Panic:         make(map[string]map[string]int64),
		EvalDuration:  make(map[string]int64),
		EvalCount:     make(map[string]int64),
		ConditionEval: make(map[string]map[string]map[string]int64),
		ActionExec:    make(map[string]map[string]map[string]int64),
	}
}

// Snapshot 返回所有字段的一致性快照（用于并发读）
func (c *MemorySink) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	snap := Snapshot{
		LoadedChains: c.LoadedChains,
		ActiveEval:   c.ActiveEval,
	}
	snap.EvalTotal = deepCopyMap2(c.EvalTotal)
	snap.ActionTotal = deepCopyMap3(c.ActionTotal)
	snap.RuleThrottled = deepCopyMap2(c.RuleThrottled)
	snap.Panic = deepCopyMap2(c.Panic)
	snap.EvalDuration = make(map[string]int64, len(c.EvalDuration))
	for k, v := range c.EvalDuration {
		snap.EvalDuration[k] = v
	}
	snap.EvalCount = make(map[string]int64, len(c.EvalCount))
	for k, v := range c.EvalCount {
		snap.EvalCount[k] = v
	}
	snap.ConditionEval = deepCopyMap3(c.ConditionEval)
	snap.ActionExec = deepCopyMap3(c.ActionExec)
	return snap
}

func deepCopyMap2(src map[string]map[string]int64) map[string]map[string]int64 {
	dst := make(map[string]map[string]int64, len(src))
	for k, v := range src {
		inner := make(map[string]int64, len(v))
		for k2, v2 := range v {
			inner[k2] = v2
		}
		dst[k] = inner
	}
	return dst
}

func deepCopyMap3(src map[string]map[string]map[string]int64) map[string]map[string]map[string]int64 {
	dst := make(map[string]map[string]map[string]int64, len(src))
	for k, v := range src {
		m2 := make(map[string]map[string]int64, len(v))
		for k2, v2 := range v {
			inner := make(map[string]int64, len(v2))
			for k3, v3 := range v2 {
				inner[k3] = v3
			}
			m2[k2] = inner
		}
		dst[k] = m2
	}
	return dst
}

func (c *MemorySink) IncEvalTotal(chainID, result string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.EvalTotal[chainID]
	if !ok {
		m = make(map[string]int64)
		c.EvalTotal[chainID] = m
	}
	m[result]++
}

func (c *MemorySink) IncActionTotal(chainID, actionType, result string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m1, ok := c.ActionTotal[chainID]
	if !ok {
		m1 = make(map[string]map[string]int64)
		c.ActionTotal[chainID] = m1
	}
	m2, ok := m1[actionType]
	if !ok {
		m2 = make(map[string]int64)
		m1[actionType] = m2
	}
	m2[result]++
}

func (c *MemorySink) IncRuleThrottled(chainID, ruleID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.RuleThrottled[chainID]
	if !ok {
		m = make(map[string]int64)
		c.RuleThrottled[chainID] = m
	}
	m[ruleID]++
}

func (c *MemorySink) IncPanic(chainID, ruleID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.Panic[chainID]
	if !ok {
		m = make(map[string]int64)
		c.Panic[chainID] = m
	}
	m[ruleID]++
}

func (c *MemorySink) ObserveEvalDuration(chainID string, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.EvalDuration[chainID] += d.Nanoseconds()
	c.EvalCount[chainID]++
}

func (c *MemorySink) SetLoadedChains(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LoadedChains = n
}

func (c *MemorySink) SetActiveEval(n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ActiveEval = n
}

func (c *MemorySink) ObserveConditionEval(chainID, nodeID string, duration time.Duration, matched bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.ConditionEval[chainID]
	if !ok {
		m = make(map[string]map[string]int64)
		c.ConditionEval[chainID] = m
	}
	n, ok := m[nodeID]
	if !ok {
		n = make(map[string]int64)
		m[nodeID] = n
	}
	n["count"]++
	n["latency_ns"] += duration.Nanoseconds()
	if matched {
		n["matched"]++
	}
}

func (c *MemorySink) ObserveActionExec(chainID, nodeID, actionType string, duration time.Duration, hasError bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.ActionExec[chainID]
	if !ok {
		m = make(map[string]map[string]int64)
		c.ActionExec[chainID] = m
	}
	n, ok := m[nodeID]
	if !ok {
		n = make(map[string]int64)
		m[nodeID] = n
	}
	n["count"]++
	n["latency_ns"] += duration.Nanoseconds()
	if hasError {
		n["error"]++
	}
}

// 编译期接口检查
var _ contract.MetricsSink = (*MemorySink)(nil)
