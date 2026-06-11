package engine

import (
	"context"
	"sync"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  Result 池（V2 3.4 内存池化）
// ─────────────────────────────────────────────

// resultPool EvalResult 对象池
//
// 设计目标：
//   - 高并发评估场景下减少 GC 压力
//   - 复用 MatchedRules/Errors 切片容量
//
// 注意事项（务必阅读）：
//   - 从池中获取的 Result 必须调用 releaseResult 归还
//   - 归还后 Result 不应再被外部使用（其内部切片可能被覆盖）
//   - 应用层若需要长期持有，应在 Release 前深拷贝所需数据
var resultPool = sync.Pool{
	New: func() interface{} {
		return &EvalResult{
			MatchedRules: make([]*core.Rule, 0, 4),
			Errors:       make([]RuleError, 0, 2),
		}
	},
}

// acquireResult 从池中获取一个 EvalResult（已重置）。
func acquireResult() *EvalResult {
	return resultPool.Get().(*EvalResult)
}

// releaseResult 归还 Result 到池中。
func releaseResult(r *EvalResult) {
	if r == nil {
		return
	}
	r.Reset()
	resultPool.Put(r)
}

// evalChainBatchPlain 非池化批量评估。
func (e *Engine) evalChainBatchPlain(ctx context.Context, chainID string, dataList []core.DataContext) ([]*EvalResult, error) {
	snap := e.snapshot.Load()
	chain, ok := snap.chains[chainID]
	if !ok {
		results := make([]*EvalResult, len(dataList))
		for i, data := range dataList {
			results[i] = &EvalResult{Data: data}
		}
		return results, nil
	}

	_ = chain

	results := make([]*EvalResult, len(dataList))
	for i, data := range dataList {
		result, _ := e.EvalChain(ctx, chainID, data)
		results[i] = result
	}
	return results, nil
}

// evalChainBatchPooled 池化批量评估。
func (e *Engine) evalChainBatchPooled(ctx context.Context, chainID string, dataList []core.DataContext) ([]*EvalResult, error) {
	results := make([]*EvalResult, len(dataList))
	for i, data := range dataList {
		result, _ := e.EvalChain(ctx, chainID, data)
		results[i] = result
	}
	return results, nil
}

// time alias to keep imports tidy
var _ = time.Now
