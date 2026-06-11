// Package main 演示多租户场景：每个租户独立的规则链 ID 前缀 + 独立限流 key。
package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/tokenbucket"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/datacontext"
)

// simpleLimiter 按 tenant 限流的薄包装
type tenantLimiter struct {
	perTenant *tokenbucket.PerKeyLimiter
}

func newTenantLimiter(rate, burst float64) *tenantLimiter {
	return &tenantLimiter{
		perTenant: tokenbucket.NewPerKeyLimiter(rate, burst),
	}
}

func (t *tenantLimiter) Allow(key string) bool {
	// key 形如 "tenant:acme:rule:r1"，前缀 tenant 决定 token bucket
	tenant := extractTenant(key)
	return t.perTenant.Allow(tenant)
}

func (t *tenantLimiter) Wait(_ context.Context, _ string) error { return nil }

func extractTenant(key string) string {
	// 简化：从 key 提取 tenant 段
	const prefix = "tenant:"
	for i := 0; i+len(prefix) <= len(key); i++ {
		if key[i:i+len(prefix)] == prefix {
			end := i + len(prefix)
			for j := end; j < len(key); j++ {
				if key[j] == ':' {
					return key[end:j]
				}
			}
			return key[end:]
		}
	}
	return "default"
}

func main() {
	limiter := newTenantLimiter(100, 20) // 每个租户 100 req/s，burst 20
	e := engine.NewEngine(engine.WithLimiter(limiter))

	// 加载 3 个租户的规则链
	for _, tenant := range []string{"acme", "globex", "initech"} {
		chain := &core.RuleChain{
			ID:   fmt.Sprintf("chain_%s", tenant),
			Root: true,
			Rules: []*core.Rule{{
				ID: "r1", Priority: 1, Enabled: true,
				Condition: &core.ConditionNode{
					Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
				},
				Actions: &core.ActionChain{},
			}},
		}
		_ = e.LoadChain(chain)
	}

	// 模拟 3 个租户并发评估
	var wg sync.WaitGroup
	for _, tenant := range []string{"acme", "globex", "initech"} {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			data := datacontext.NewMapDataContext(map[string]any{
				"value": float64(42),
				"tags":  map[string]string{"tenant": t},
			})
			// 模拟该租户的高频请求
			for i := 0; i < 50; i++ {
				_, _ = e.EvalChain(context.Background(), fmt.Sprintf("chain_%s", t), data)
			}
		}(tenant)
	}
	wg.Wait()

	fmt.Println("multi-tenant eval done")
}
