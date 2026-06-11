/**
 * MultiInputBuffer — 多输入缓冲区（Phase 5 新增）
 *
 * 用于缓存单点数据，待齐集后触发多输入规则评估
 *
 * 设计要点：
 * - 按设备分组缓存多输入上下文
 * - 支持超时清理，避免内存泄漏
 * - 支持触发回调，通知规则引擎评估
 * - 零分配设计：使用池化 MultiDataContext
 */

package datacontext

import (
	"context"
	"sync"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  MultiInputBuffer — 多输入缓冲区
// ─────────────────────────────────────────────

// MultiInputBuffer 多输入缓冲区
type MultiInputBuffer struct {
	// 缓存：deviceID → pointName → MultiDataContext
	// 同一设备的多个输入点共享同一个 MultiDataContext
	entries map[string]map[string]*MultiDataContext // deviceID → pointName → MDC

	// 触发回调
	triggerCallback func(ctx context.Context, mdc *MultiDataContext)

	// 超时配置
	timeout       time.Duration // 超时时间（默认 5s）
	cleanupTicker *time.Ticker  // 清理定时器

	// 池
	pool *MultiDataContextPool

	// 锁
	mu sync.RWMutex

	// 上下文
	ctx context.Context
}

// NewMultiInputBuffer 创建多输入缓冲区
func NewMultiInputBuffer(ctx context.Context, timeout time.Duration) *MultiInputBuffer {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	buf := &MultiInputBuffer{
		entries: make(map[string]map[string]*MultiDataContext),
		timeout: timeout,
		pool:    NewMultiDataContextPool(),
		ctx:     ctx,
	}

	// 启动清理定时器
	buf.cleanupTicker = time.NewTicker(timeout)
	go buf.cleanupLoop()

	return buf
}

// SetTriggerCallback 设置触发回调
func (buf *MultiInputBuffer) SetTriggerCallback(callback func(ctx context.Context, mdc *MultiDataContext)) {
	buf.triggerCallback = callback
}

// Add 添加数据点
// 返回是否触发评估（齐集完成）
func (buf *MultiInputBuffer) Add(chainID string, inputNames []string, data core.DataContext) bool {
	buf.mu.Lock()
	defer buf.mu.Unlock()

	deviceID := data.DeviceID()
	pointName := data.PointName()

	// 检查是否为声明的输入
	if !isDeclaredInput(pointName, inputNames) {
		return false
	}

	// 获取或创建设备缓存
	deviceCache, ok := buf.entries[deviceID]
	if !ok {
		deviceCache = make(map[string]*MultiDataContext)
		buf.entries[deviceID] = deviceCache
	}

	// 获取或创建 MultiDataContext（按 chainID + deviceID）
	// 使用 chainID 作为 key，确保同一规则链的同一设备共享一个 MDC
	mdcKey := chainID
	mdc, ok := deviceCache[mdcKey]
	if !ok {
		// 创建新的 MultiDataContext
		mdc = buf.pool.Acquire(buf.ctx, deviceID, chainID, inputNames)
		deviceCache[mdcKey] = mdc
	}

	// 添加数据点
	ready := mdc.AddPoint(pointName, data)

	// 如果齐集完成，触发回调
	if ready && buf.triggerCallback != nil {
		// 异步触发回调
		go buf.triggerCallback(buf.ctx, mdc)

		// 清理缓存
		delete(deviceCache, mdcKey)
		if len(deviceCache) == 0 {
			delete(buf.entries, deviceID)
		}

		// 释放 MultiDataContext
		buf.pool.Release(mdc)
	}

	return ready
}

// Get 获取设备的 MultiDataContext
func (buf *MultiInputBuffer) Get(deviceID string) map[string]*MultiDataContext {
	buf.mu.RLock()
	defer buf.mu.RUnlock()

	return buf.entries[deviceID]
}

// Remove 移除设备的缓存
func (buf *MultiInputBuffer) Remove(deviceID string) {
	buf.mu.Lock()
	defer buf.mu.Unlock()

	deviceCache, ok := buf.entries[deviceID]
	if !ok {
		return
	}

	// 释放所有 MultiDataContext
	for _, mdc := range deviceCache {
		buf.pool.Release(mdc)
	}

	delete(buf.entries, deviceID)
}

// Clear 清空所有缓存
func (buf *MultiInputBuffer) Clear() {
	buf.mu.Lock()
	defer buf.mu.Unlock()

	// 释放所有 MultiDataContext
	for _, deviceCache := range buf.entries {
		for _, mdc := range deviceCache {
			buf.pool.Release(mdc)
		}
	}

	buf.entries = make(map[string]map[string]*MultiDataContext)
}

// Size 返回缓存大小
func (buf *MultiInputBuffer) Size() int {
	buf.mu.RLock()
	defer buf.mu.RUnlock()

	count := 0
	for _, deviceCache := range buf.entries {
		count += len(deviceCache)
	}
	return count
}

// Stop 停止缓冲区
func (buf *MultiInputBuffer) Stop() {
	buf.cleanupTicker.Stop()
	buf.Clear()
}

// ─────────────────────────────────────────────
//  清理循环
// ─────────────────────────────────────────────

func (buf *MultiInputBuffer) cleanupLoop() {
	for {
		select {
		case <-buf.cleanupTicker.C:
			buf.cleanup()
		case <-buf.ctx.Done():
			return
		}
	}
}

func (buf *MultiInputBuffer) cleanup() {
	buf.mu.Lock()
	defer buf.mu.Unlock()

	now := time.Now().UnixNano()

	// 清理超时的缓存
	for deviceID, deviceCache := range buf.entries {
		for pointName, mdc := range deviceCache {
			// 检查是否超时
			if now-mdc.Timestamp > int64(buf.timeout) {
				// 释放 MultiDataContext
				buf.pool.Release(mdc)
				delete(deviceCache, pointName)
			}
		}

		// 如果设备缓存为空，删除设备
		if len(deviceCache) == 0 {
			delete(buf.entries, deviceID)
		}
	}
}

// ─────────────────────────────────────────────
//  辅助函数
// ─────────────────────────────────────────────

func isDeclaredInput(pointName string, inputNames []string) bool {
	for _, name := range inputNames {
		if name == pointName {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
//  MultiInputBuffer 配置
// ─────────────────────────────────────────────

// MultiInputBufferConfig 多输入缓冲区配置
type MultiInputBufferConfig struct {
	// 超时时间（默认 5s）
	Timeout time.Duration

	// 最大缓存大小（默认 10000）
	MaxSize int

	// 触发回调
	TriggerCallback func(ctx context.Context, mdc *MultiDataContext)
}

// NewMultiInputBufferWithConfig 创建多输入缓冲区（带配置）
func NewMultiInputBufferWithConfig(ctx context.Context, config MultiInputBufferConfig) *MultiInputBuffer {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	buf := NewMultiInputBuffer(ctx, config.Timeout)

	if config.TriggerCallback != nil {
		buf.SetTriggerCallback(config.TriggerCallback)
	}

	return buf
}
