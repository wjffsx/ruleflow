// Package router 提供数据路由策略实现。
//
// V12 架构定位说明：
//   - 本包是可选的支撑层模块，不属于核心库
//   - 用于根据 pipelineType 和输入声明路由数据到对应的规则链
//   - 依赖方向：router → core（正确）
//
// 职责边界：
//   - 核心库（core/engine）：提供规则评估能力，不包含路由逻辑
//   - router：提供数据路由策略，可选启用
//   - 应用层：决定是否使用路由器，组合 engine + router
//
// 使用场景：
//   - 多链场景：根据数据点类型和声明路由到不同规则链
//   - Phase 1 输入声明：支持规则链的输入声明和类型匹配
//
// 不使用场景：
//   - 单链场景：直接调用 engine.EvalChain，无需路由器
//   - 简单场景：数据点类型固定，无需路由策略
package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  DataRouter — 数据路由器（Phase 1 新增）
// ─────────────────────────────────────────────

// DataRouter 根据 pipelineType 和输入声明路由数据到对应的规则链
type DataRouter struct {
	entries map[string]*RouterEntry // chainID → 路由条目
	mu      sync.RWMutex
}

// RouterEntry 路由条目
type RouterEntry struct {
	ChainID      string
	PipelineType string         // "analog" | "digital" | "meter" | ""
	InputIndex   map[string]int // point_name → index 的倒排索引
	InputCount   int            // 输入数量
}

// NewDataRouter 创建数据路由器
func NewDataRouter() *DataRouter {
	return &DataRouter{
		entries: make(map[string]*RouterEntry),
	}
}

// RegisterChain 注册规则链到路由器
func (r *DataRouter) RegisterChain(chain *core.RuleChain) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 构建输入索引
	inputIndex := make(map[string]int)
	for i, inp := range chain.Inputs {
		inputIndex[inp.PointName] = i
	}

	r.entries[chain.ID] = &RouterEntry{
		ChainID:      chain.ID,
		PipelineType: chain.PipelineType,
		InputIndex:   inputIndex,
		InputCount:   len(chain.Inputs),
	}

	return nil
}

// UnregisterChain 从路由器移除规则链
func (r *DataRouter) UnregisterChain(chainID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, chainID)
}

// RouteResult 路由结果
type RouteResult struct {
	ChainID     string
	InputIndex  int  // 数据点在链输入中的索引
	MatchedType bool // pipelineType 是否匹配
	Declared    bool // 数据点是否在输入声明中
}

// Route 路由数据点到规则链
// 返回匹配的链 ID 和路由详情
func (r *DataRouter) Route(ctx context.Context, data core.DataContext) ([]RouteResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []RouteResult

	for _, entry := range r.entries {
		// 1. pipelineType 匹配检查
		typeMatched := true
		if entry.PipelineType != "" && entry.PipelineType != data.PointType() {
			typeMatched = false
		}

		// 2. 输入声明检查
		idx, declared := entry.InputIndex[data.PointName()]

		// 只返回完全匹配的链
		if typeMatched && declared {
			results = append(results, RouteResult{
				ChainID:     entry.ChainID,
				InputIndex:  idx,
				MatchedType: typeMatched,
				Declared:    declared,
			})
		}
	}

	return results, nil
}

// RouteToChain 路由数据点到指定规则链
// 用于已知目标链的场景，返回是否匹配
func (r *DataRouter) RouteToChain(ctx context.Context, chainID string, data core.DataContext) (RouteResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[chainID]
	if !ok {
		return RouteResult{}, fmt.Errorf("unknown chain: %s", chainID)
	}

	// 1. pipelineType 匹配检查
	typeMatched := true
	if entry.PipelineType != "" && entry.PipelineType != data.PointType() {
		typeMatched = false
	}

	// 2. 输入声明检查
	idx, declared := entry.InputIndex[data.PointName()]

	return RouteResult{
		ChainID:     entry.ChainID,
		InputIndex:  idx,
		MatchedType: typeMatched,
		Declared:    declared,
	}, nil
}

// GetChainIDs 返回已注册的链 ID 列表
func (r *DataRouter) GetChainIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.entries))
	for id := range r.entries {
		ids = append(ids, id)
	}
	return ids
}

// GetRouterEntry 返回指定链的路由条目（用于调试）
func (r *DataRouter) GetRouterEntry(chainID string) (*RouterEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[chainID]
	return entry, ok
}

// ─────────────────────────────────────────────
//  路由错误类型
// ─────────────────────────────────────────────

// ErrTypeMismatch 类型不匹配错误
var ErrTypeMismatch = fmt.Errorf("pipeline type mismatch")

// ErrUndeclaredInput 未声明输入错误
var ErrUndeclaredInput = fmt.Errorf("undeclared input point")

// RouteError 路由错误详情
type RouteError struct {
	ChainID      string
	PointName    string
	PointType    string
	PipelineType string
	Declared     bool
}

func (e *RouteError) Error() string {
	if !e.Declared {
		return fmt.Sprintf("undeclared input %q for chain %s", e.PointName, e.ChainID)
	}
	return fmt.Sprintf("type mismatch: chain=%s, data=%s", e.PipelineType, e.PointType)
}

// ValidateRoute 校验路由是否有效（返回详细错误）
func (r *DataRouter) ValidateRoute(ctx context.Context, chainID string, data core.DataContext) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[chainID]
	if !ok {
		return fmt.Errorf("unknown chain: %s", chainID)
	}

	// 1. pipelineType 匹配检查
	if entry.PipelineType != "" && entry.PipelineType != data.PointType() {
		return &RouteError{
			ChainID:      entry.ChainID,
			PointName:    data.PointName(),
			PointType:    data.PointType(),
			PipelineType: entry.PipelineType,
			Declared:     true,
		}
	}

	// 2. 输入声明检查
	if _, declared := entry.InputIndex[data.PointName()]; !declared {
		return &RouteError{
			ChainID:      entry.ChainID,
			PointName:    data.PointName(),
			PointType:    data.PointType(),
			PipelineType: entry.PipelineType,
			Declared:     false,
		}
	}

	return nil
}
