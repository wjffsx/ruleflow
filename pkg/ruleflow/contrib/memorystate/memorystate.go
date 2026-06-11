// Package memorystate 提供 ruleflow 引擎的内存版 StateStore 实现。
//
// 设计要点：
//   - 线程安全：基于 sync.Map，所有方法可在并发热路径上调用
//   - 零依赖：仅使用 stdlib sync.Map
//
// 基本用法：
//
//	import "github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/memorystate"
//
//	store := memorystate.NewMapStateStore()
//	eng := engine.NewEngine(engine.WithStateStore(store))
package memorystate

import "sync"

// MapStateStore 基于 sync.Map 的内存状态存储
type MapStateStore struct {
	data sync.Map
}

// NewMapStateStore 创建基于 sync.Map 的状态存储
func NewMapStateStore() *MapStateStore {
	return &MapStateStore{}
}

// Get 返回 key 对应的值与存在标志
func (s *MapStateStore) Get(key string) (any, bool) { return s.data.Load(key) }

// Set 设置 key 对应的值
func (s *MapStateStore) Set(key string, value any) { s.data.Store(key, value) }

// Delete 删除 key
func (s *MapStateStore) Delete(key string) { s.data.Delete(key) }

// 编译期接口检查（state 包若有 StateStore 接口）
// var _ state.StateStore = (*MapStateStore)(nil)
