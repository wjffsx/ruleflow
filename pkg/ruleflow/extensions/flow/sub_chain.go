// Package flow provides VPP flow nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package flow

import (
	"context"
	"fmt"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  SubChainAction — 子规则链调用动作
// ─────────────────────────────────────────────

// SubChainAction 子规则链调用动作
type SubChainAction struct {
	IDValue       string            `json:"id"`
	ChainID       string            `json:"chain_id"`
	Sync          bool              `json:"sync"`
	InputMapping  map[string]string `json:"input_mapping"`
	OutputMapping map[string]string `json:"output_mapping"`

	// 运行时注入
	Engine types.SubChainEngine
}

// NewSubChainAction 创建子规则链调用动作
func NewSubChainAction(id, chainID string, sync bool, inputMapping, outputMapping map[string]string, engine types.SubChainEngine) *SubChainAction {
	return &SubChainAction{
		IDValue:       id,
		ChainID:       chainID,
		Sync:          sync,
		InputMapping:  inputMapping,
		OutputMapping: outputMapping,
		Engine:        engine,
	}
}

func (a *SubChainAction) Execute(ctx context.Context, data core.DataContext) error {
	if a.Engine == nil {
		return fmt.Errorf("sub_chain: no SubChainEngine configured")
	}

	// 1. 输入映射
	inputCtx := data
	if len(a.InputMapping) > 0 {
		// 创建映射后的数据上下文
		mappedData := NewMappedDataContext(data, a.InputMapping)
		inputCtx = mappedData
	}

	// 2. 执行子链
	var err error
	if a.Sync {
		err = a.Engine.ExecuteChain(ctx, a.ChainID, inputCtx)
	} else {
		go func() {
			_ = a.Engine.ExecuteChain(context.Background(), a.ChainID, inputCtx)
		}()
	}

	if err != nil {
		data.SetTag("_sub_chain_status", "failed")
		return fmt.Errorf("sub_chain %s: %w", a.ChainID, err)
	}

	// 3. 输出映射
	if len(a.OutputMapping) > 0 && a.Sync {
		for childKey, parentKey := range a.OutputMapping {
			if val := inputCtx.GetTag(childKey); val != "" {
				data.SetTag(parentKey, val)
			}
		}
	}

	data.SetTag("_sub_chain_id", a.ChainID)
	data.SetTag("_sub_chain_status", "ok")
	return nil
}

func (a *SubChainAction) ID() string          { return a.IDValue }
func (a *SubChainAction) Type() string        { return "sub_chain" }
func (a *SubChainAction) Description() string { return fmt.Sprintf("sub chain call to %s", a.ChainID) }

// ─────────────────────────────────────────────
//  MappedDataContext — 映射数据上下文
// ─────────────────────────────────────────────

// MappedDataContext 映射数据上下文（用于子链调用）
type MappedDataContext struct {
	parent       core.DataContext
	inputMapping map[string]string // parentKey → childKey
	tagCache     map[string]string
}

// NewMappedDataContext 创建映射数据上下文
func NewMappedDataContext(parent core.DataContext, inputMapping map[string]string) *MappedDataContext {
	return &MappedDataContext{
		parent:       parent,
		inputMapping: inputMapping,
		tagCache:     make(map[string]string),
	}
}

func (m *MappedDataContext) DeviceID() string            { return m.parent.DeviceID() }
func (m *MappedDataContext) PointName() string           { return m.parent.PointName() }
func (m *MappedDataContext) SetPointName(name string)    { m.parent.SetPointName(name) }
func (m *MappedDataContext) PointType() string           { return m.parent.PointType() }
func (m *MappedDataContext) FQN() string                 { return m.parent.FQN() }
func (m *MappedDataContext) Value() float64              { return m.parent.Value() }
func (m *MappedDataContext) SetValue(v float64)          { m.parent.SetValue(v) }
func (m *MappedDataContext) Quality() int                { return m.parent.Quality() }
func (m *MappedDataContext) SetQuality(q int)            { m.parent.SetQuality(q) }
func (m *MappedDataContext) UpperLimit() (float64, bool) { return m.parent.UpperLimit() }
func (m *MappedDataContext) LowerLimit() (float64, bool) { return m.parent.LowerLimit() }
func (m *MappedDataContext) LimitExceeded() bool         { return m.parent.LimitExceeded() }
func (m *MappedDataContext) SetLimitExceeded(v bool)     { m.parent.SetLimitExceeded(v) }
func (m *MappedDataContext) GetTag(key string) string {
	if val, ok := m.tagCache[key]; ok {
		return val
	}
	return m.parent.GetTag(key)
}
func (m *MappedDataContext) SetTag(key, value string) {
	m.tagCache[key] = value
	m.parent.SetTag(key, value)
}
func (m *MappedDataContext) TargetCount() int                       { return m.parent.TargetCount() }
func (m *MappedDataContext) TargetAt(i int) string                  { return m.parent.TargetAt(i) }
func (m *MappedDataContext) AddTarget(target string)                { m.parent.AddTarget(target) }
func (m *MappedDataContext) Dropped() bool                          { return m.parent.Dropped() }
func (m *MappedDataContext) SetDropped(v bool)                      { m.parent.SetDropped(v) }
func (m *MappedDataContext) Timestamp() int64                       { return m.parent.Timestamp() }
func (m *MappedDataContext) SpanContext() contract.SpanContext      { return m.parent.SpanContext() }
func (m *MappedDataContext) SetSpanContext(sc contract.SpanContext) { m.parent.SetSpanContext(sc) }
func (m *MappedDataContext) PreviousValue() (float64, bool)         { return m.parent.PreviousValue() }
func (m *MappedDataContext) SetPreviousValue(v float64)             { m.parent.SetPreviousValue(v) }
func (m *MappedDataContext) Raw() any                               { return m.parent.Raw() }

// MultiDataContext 接口实现
func (m *MappedDataContext) GetPoint(pointName string) (float64, error) {
	if mdc, ok := m.parent.(types.MultiDataContextInterface); ok {
		return mdc.GetPoint(pointName)
	}
	return 0, fmt.Errorf("point %s not found", pointName)
}

func (m *MappedDataContext) GetPointData(pointName string) (core.DataContext, bool) {
	if mdc, ok := m.parent.(types.MultiDataContextInterface); ok {
		return mdc.GetPointData(pointName)
	}
	return nil, false
}

func (m *MappedDataContext) GetAllPoints() []string {
	if mdc, ok := m.parent.(types.MultiDataContextInterface); ok {
		return mdc.GetAllPoints()
	}
	return nil
}

func (m *MappedDataContext) SetPointValue(pointName string, value float64) {
	if mdc, ok := m.parent.(types.MultiDataContextInterface); ok {
		mdc.SetPointValue(pointName, value)
	}
}

// StatefulDataContext 接口实现
func (m *MappedDataContext) StateStore() core.StateStore {
	if sdc, ok := m.parent.(core.StatefulDataContext); ok {
		return sdc.StateStore()
	}
	return nil
}
