package engine

import "github.com/wjffsx/ruleflow/pkg/ruleflow/core"

// ─────────────────────────────────────────────
//  WrapWithStateStore — DataContext 包装函数
// ─────────────────────────────────────────────

// statefulDataWrapper 将 DataContext 包装为 StatefulDataContext
type statefulDataWrapper struct {
	core.DataContext
	store core.StateStore
}

func (w *statefulDataWrapper) StateStore() core.StateStore { return w.store }

// WrapWithStateStore 将 DataContext 包装为 StatefulDataContext
//
// V4.2：SyncMapStateStore 已迁出至 contrib/memorystate.MapStateStore。
// 引擎仅保留 DataContext 包装职责（与 state 实现解耦）。
func WrapWithStateStore(data core.DataContext, store core.StateStore) core.StatefulDataContext {
	return &statefulDataWrapper{DataContext: data, store: store}
}
