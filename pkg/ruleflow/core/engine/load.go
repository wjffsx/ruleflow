package engine

import (
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/compiler"
)

// loadChain 加载并编译规则链（原子交换）。
func (e *Engine) loadChain(chain *core.RuleChain) error {
	compiled, err := compiler.CompileChain(chain, e.registry)
	if err != nil {
		e.logger.Error("load chain failed: compile error", "chain_id", chain.ID, "err", err)
		return core.NewConfigError(chain.ID, err)
	}
	compiled.ErrorStrategy = e.errorStrategy
	compiled.EvalMode = e.evalMode

	e.writeMu.Lock()
	defer e.writeMu.Unlock()

	// 添加节点并检测循环依赖
	e.depGraph.AddNode(chain.ID)
	for _, ref := range chain.Refs {
		if err := e.depGraph.AddEdge(chain.ID, ref); err != nil {
			e.logger.Warn("load chain rejected: cyclic dependency",
				"chain_id", chain.ID, "ref", ref, "err", err)
			return err
		}
	}

	// COW: 复制当前快照，替换目标链
	old := e.snapshot.Load()
	newSnap := &chainSnapshot{
		chains: make(map[string]*core.CompiledChain, len(old.chains)),
		rules:  make(map[string]*core.CompiledRule, len(old.rules)),
	}
	for k, v := range old.chains {
		newSnap.chains[k] = v
	}
	for k, v := range old.rules {
		newSnap.rules[k] = v
	}
	newSnap.chains[chain.ID] = compiled
	for _, cr := range compiled.SortedRules {
		newSnap.rules[cr.Rule.ID] = cr
	}

	e.snapshot.Store(newSnap) // 原子交换
	e.metricsSink.SetLoadedChains(len(newSnap.chains))
	e.logger.Info("chain loaded",
		"chain_id", chain.ID,
		"version", compiled.Version,
		"rules", len(compiled.SortedRules),
		"refs", chain.Refs)
	return nil
}

// unloadChain 卸载规则链。
func (e *Engine) unloadChain(chainID string) {
	e.writeMu.Lock()
	defer e.writeMu.Unlock()

	old := e.snapshot.Load()
	newSnap := &chainSnapshot{
		chains: make(map[string]*core.CompiledChain, len(old.chains)),
		rules:  make(map[string]*core.CompiledRule),
	}
	for k, v := range old.chains {
		if k != chainID {
			newSnap.chains[k] = v
		}
	}
	// 仅保留属于其他链的规则
	chainRules := make(map[string]struct{})
	if cc, ok := old.chains[chainID]; ok {
		for _, cr := range cc.SortedRules {
			chainRules[cr.Rule.ID] = struct{}{}
		}
	}
	for k, v := range old.rules {
		if _, skip := chainRules[k]; !skip {
			newSnap.rules[k] = v
		}
	}

	e.snapshot.Store(newSnap)

	e.depGraph.RemoveNode(chainID)

	e.metricsSink.SetLoadedChains(len(newSnap.chains))
	e.logger.Info("chain unloaded", "chain_id", chainID)
}

// getChainIDs 返回已加载的链 ID 列表。
func (e *Engine) getChainIDs() []string {
	snap := e.snapshot.Load()
	ids := make([]string, 0, len(snap.chains))
	for id := range snap.chains {
		ids = append(ids, id)
	}
	return ids
}