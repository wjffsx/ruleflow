/**
 * MultiDataContext — 多输入数据上下文（Phase 5 新增）
 *
 * 用于需要多输入聚合的规则，如：
 * - 多点平均值计算
 * - 多点逻辑组合
 * - 多点趋势分析
 *
 * 设计要点：
 * - 缓存单点数据，待齐集后触发规则评估
 * - 支持超时机制，避免永久等待
 * - 零分配设计：预分配缓冲区，避免堆分配
 */

package datacontext

import (
	"context"
	"sync"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  MultiDataContext — 多输入数据上下文
// ─────────────────────────────────────────────

// MultiDataContext 多输入数据上下文
// 用于需要多输入聚合的规则
type MultiDataContext struct {
	// 基础标识
	DeviceID string
	ChainID  string

	// 输入数据点
	Points map[string]core.DataContext // point_name → DataContext

	// 齐集状态
	Ready map[string]bool // point_name → 是否已收到数据
	Count int             // 已齐集的数据点数量
	Total int             // 需要齐集的总数据点数量

	// 元数据
	Timestamp int64           // 最后更新时间戳
	mu        sync.RWMutex    // 读写锁
	ctx       context.Context // 上下文
}

// NewMultiDataContext 创建多输入数据上下文
func NewMultiDataContext(ctx context.Context, deviceID, chainID string, inputNames []string) *MultiDataContext {
	mdc := &MultiDataContext{
		DeviceID:  deviceID,
		ChainID:   chainID,
		Points:    make(map[string]core.DataContext, len(inputNames)),
		Ready:     make(map[string]bool, len(inputNames)),
		Total:     len(inputNames),
		Count:     0,
		Timestamp: time.Now().UnixNano(),
		ctx:       ctx,
	}

	// 初始化 Ready 状态
	for _, name := range inputNames {
		mdc.Ready[name] = false
	}

	return mdc
}

// AddPoint 添加数据点
// 返回是否齐集完成
func (mdc *MultiDataContext) AddPoint(pointName string, data core.DataContext) bool {
	mdc.mu.Lock()
	defer mdc.mu.Unlock()

	// 检查是否为声明的输入
	if !mdc.isDeclared(pointName) {
		return false
	}

	// 检查是否已齐集
	if mdc.Ready[pointName] {
		// 更新数据点
		mdc.Points[pointName] = data
		mdc.Timestamp = time.Now().UnixNano()
		return mdc.Count == mdc.Total
	}

	// 添加数据点
	mdc.Points[pointName] = data
	mdc.Ready[pointName] = true
	mdc.Count++
	mdc.Timestamp = time.Now().UnixNano()

	// 检查是否齐集完成
	return mdc.Count == mdc.Total
}

// IsReady 检查是否齐集完成
func (mdc *MultiDataContext) IsReady() bool {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	return mdc.Count == mdc.Total
}

// GetPoint 获取数据点
func (mdc *MultiDataContext) GetPoint(pointName string) (core.DataContext, bool) {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	data, ok := mdc.Points[pointName]
	return data, ok
}

// GetAllPoints 获取所有数据点
func (mdc *MultiDataContext) GetAllPoints() map[string]core.DataContext {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	// 返回副本
	result := make(map[string]core.DataContext, len(mdc.Points))
	for k, v := range mdc.Points {
		result[k] = v
	}
	return result
}

// GetMissingInputs 获取未齐集的输入点
func (mdc *MultiDataContext) GetMissingInputs() []string {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	var missing []string
	for name, ready := range mdc.Ready {
		if !ready {
			missing = append(missing, name)
		}
	}
	return missing
}

// GetReadyInputs 获取已齐集的输入点
func (mdc *MultiDataContext) GetReadyInputs() []string {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	var ready []string
	for name, r := range mdc.Ready {
		if r {
			ready = append(ready, name)
		}
	}
	return ready
}

// Reset 重置齐集状态
func (mdc *MultiDataContext) Reset() {
	mdc.mu.Lock()
	defer mdc.mu.Unlock()

	// 清空数据点
	for name := range mdc.Points {
		mdc.Points[name] = nil
		mdc.Ready[name] = false
	}
	mdc.Count = 0
	mdc.Timestamp = time.Now().UnixNano()
}

// Age 返回上下文年龄（纳秒）
func (mdc *MultiDataContext) Age() int64 {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	return time.Now().UnixNano() - mdc.Timestamp
}

// Context 返回上下文
func (mdc *MultiDataContext) Context() context.Context {
	return mdc.ctx
}

// isDeclared 检查是否为声明的输入
func (mdc *MultiDataContext) isDeclared(pointName string) bool {
	_, ok := mdc.Ready[pointName]
	return ok
}

// ─────────────────────────────────────────────
//  MultiDataContext 辅助方法
// ─────────────────────────────────────────────

// AverageValue 计算平均值
func (mdc *MultiDataContext) AverageValue() float64 {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	if mdc.Count == 0 {
		return 0
	}

	var sum float64
	var count int
	for _, data := range mdc.Points {
		if data != nil {
			sum += data.Value()
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}

// MaxValue 计算最大值
func (mdc *MultiDataContext) MaxValue() float64 {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	var max float64
	var hasValue bool
	for _, data := range mdc.Points {
		if data != nil {
			v := data.Value()
			if !hasValue || v > max {
				max = v
				hasValue = true
			}
		}
	}
	return max
}

// MinValue 计算最小值
func (mdc *MultiDataContext) MinValue() float64 {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	var min float64
	var hasValue bool
	for _, data := range mdc.Points {
		if data != nil {
			v := data.Value()
			if !hasValue || v < min {
				min = v
				hasValue = true
			}
		}
	}
	return min
}

// AllTrue 检查所有布尔值是否为 true
func (mdc *MultiDataContext) AllTrue() bool {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	for _, data := range mdc.Points {
		if data == nil || data.Value() == 0 {
			return false
		}
	}
	return true
}

// AnyTrue 检查是否有布尔值为 true
func (mdc *MultiDataContext) AnyTrue() bool {
	mdc.mu.RLock()
	defer mdc.mu.RUnlock()

	for _, data := range mdc.Points {
		if data != nil && data.Value() != 0 {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
//  MultiDataContext 池（零分配设计）
// ─────────────────────────────────────────────

// MultiDataContextPool 多输入数据上下文池
type MultiDataContextPool struct {
	pool sync.Pool
}

// NewMultiDataContextPool 创建池
func NewMultiDataContextPool() *MultiDataContextPool {
	return &MultiDataContextPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &MultiDataContext{
					Points: make(map[string]core.DataContext),
					Ready:  make(map[string]bool),
				}
			},
		},
	}
}

// Acquire 获取上下文
func (p *MultiDataContextPool) Acquire(ctx context.Context, deviceID, chainID string, inputNames []string) *MultiDataContext {
	mdc := p.pool.Get().(*MultiDataContext)

	// 重置并初始化
	mdc.DeviceID = deviceID
	mdc.ChainID = chainID
	mdc.ctx = ctx
	mdc.Count = 0
	mdc.Total = len(inputNames)
	mdc.Timestamp = time.Now().UnixNano()

	// 清空并初始化 Points 和 Ready
	for k := range mdc.Points {
		delete(mdc.Points, k)
	}
	for k := range mdc.Ready {
		delete(mdc.Ready, k)
	}
	for _, name := range inputNames {
		mdc.Points[name] = nil
		mdc.Ready[name] = false
	}

	return mdc
}

// Release 释放上下文
func (p *MultiDataContextPool) Release(mdc *MultiDataContext) {
	// 清空数据点
	for k := range mdc.Points {
		mdc.Points[k] = nil
	}
	mdc.Count = 0
	p.pool.Put(mdc)
}
