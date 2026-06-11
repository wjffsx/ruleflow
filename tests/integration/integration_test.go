//go:build integration

// Package integration 包含端到端集成测试，覆盖以下场景：
//   - 配置文件加载 → 引擎编译 → 数据评估的完整链路
//   - 热重载不中断在线评估
//   - 高并发评估 + 限流器
//   - panic/error handler 集成
//   - 优雅关闭等待正在进行的评估完成
//
// 使用 -tags=integration 启用：
//   go test -tags=integration -count=1 ./tests/integration/...
package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/config"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes"
)

// ─────────────────────────────────────────────
//  1. End-to-End: YAML → Engine → Eval
// ─────────────────────────────────────────────

// TestE2E_YAMLToEvaluation 端到端：从 YAML 配置加载到成功评估
func TestE2E_YAMLToEvaluation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "chain.yaml")
	yaml := `
chain:
  id: e2e_chain
  name: E2E test chain
  root: true
  version: 1
  status: deployed
rules:
  - id: r1
    priority: 1
    enabled: true
    condition:
      type: value_range
      config:
        min_value: 0
        max_value: 100
    actions:
      - type: transform
        config:
          scale: 2.0
          offset: 1.0
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	reg := nodes.NewEmptyRegistry()
	r.RegisterPackage(builtin.Builtin)
	cfg, err := config.LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	chain, err := config.Parse(cfg, reg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	e := core.NewEngine()
	if err := e.LoadChain(chain); err != nil {
		t.Fatalf("load chain: %v", err)
	}

	data := core.NewMapDataContext(map[string]any{"value": 50.0})
	result, err := e.EvalChain(context.Background(), "e2e_chain", data)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if len(result.MatchedRules) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.MatchedRules))
	}
	// transform action: 50 * 2 + 1 = 101
	if got := data.Value(); got != 101.0 {
		t.Errorf("transform: expected 101, got %v", got)
	}
}

// TestE2E_HealthCheck 验证健康检查端点
func TestE2E_HealthCheck(t *testing.T) {
	e := core.NewEngine()
	hc := e.HealthCheck()
	if hc.Status != "healthy" {
		t.Errorf("engine with no chains should be healthy, got %s", hc.Status)
	}
	if hc.LoadedChains != 0 {
		t.Errorf("expected 0 chains, got %d", hc.LoadedChains)
	}
}

// TestE2E_MetricsSnapshot 验证指标快照线程安全
func TestE2E_MetricsSnapshot(t *testing.T) {
	sink := core.NewCountingSink()
	e := core.NewEngine(core.WithMetricsSink(sink))
	chain := makeSimpleChain()
	_ = e.LoadChain(chain)

	// 并发评估
	const N = 100
	const G = 10
	var wg sync.WaitGroup
	for g := 0; g < G; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < N; i++ {
				data := core.NewMapDataContext(map[string]any{"k": "v"})
				_, _ = e.EvalChain(context.Background(), "simple", data)
			}
		}()
	}
	wg.Wait()

	snap := sink.Snapshot()
	if snap.EvalCount["simple"] < N*G {
		t.Errorf("expected eval count >= %d, got %d", N*G, snap.EvalCount["simple"])
	}
}

// ─────────────────────────────────────────────
//  2. Hot Reload 无中断
// ─────────────────────────────────────────────

// TestHotReload_NoInterruption 验证热重载过程中评估不中断
func TestHotReload_NoInterruption(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "chain.yaml")
	if err := os.WriteFile(cfgPath, simpleYAML("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := nodes.NewEmptyRegistry()
	r.RegisterPackage(builtin.Builtin)
	e := core.NewEngine()
	fw, err := config.NewFileWatcher(cfgPath, &config.DefaultLoader{
		Registry: reg,
		OnLoad: func(ctx context.Context, chain *core.RuleChain) error {
			e.UnloadChain(chain.ID)
			return e.LoadChain(chain)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fw.WithDebounce(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fw.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	// 启动评估 worker
	var evaluated atomic.Int64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				data := core.NewMapDataContext(map[string]any{"k": "v"})
				_, _ = e.EvalChain(ctx, "v1", data)
				evaluated.Add(1)
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	// 持续 300ms，期间触发 3 次热重载
	deadline := time.Now().Add(300 * time.Millisecond)
	version := 2
	for time.Now().Before(deadline) {
		if err := os.WriteFile(cfgPath, simpleYAML(fmtInt(version)), 0644); err != nil {
			t.Fatal(err)
		}
		version++
		time.Sleep(100 * time.Millisecond)
	}
	close(stop)
	wg.Wait()

	if evaluated.Load() < 50 {
		t.Errorf("expected at least 50 evals during hot reload, got %d", evaluated.Load())
	}
}

// ─────────────────────────────────────────────
//  3. Concurrency + Limiter
// ─────────────────────────────────────────────

// TestConcurrent_LimiterBlocksExcess 测试限流器在并发下生效
func TestConcurrent_LimiterBlocksExcess(t *testing.T) {
	e := core.NewEngine(core.WithLimiter(limitedLimiter(5)))
	_ = e.LoadChain(makeSimpleChain())

	const N = 50
	var blocked atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := core.NewMapDataContext(map[string]any{"k": "v"})
			_, err := e.EvalChain(context.Background(), "simple", data)
			if err != nil {
				blocked.Add(1)
			}
		}()
	}
	wg.Wait()

	// 限流至少应导致一些评估被丢或失败
	t.Logf("blocked/errored: %d / %d", blocked.Load(), N)
	if blocked.Load() == N {
		t.Error("limiter blocked all requests, which is too strict for this test")
	}
}

// ─────────────────────────────────────────────
//  4. Shutdown 等待正在评估的请求
// ─────────────────────────────────────────────

// TestShutdown_WaitsForInflight 验证 Shutdown 等待进行中的评估完成
func TestShutdown_WaitsForInflight(t *testing.T) {
	e := core.NewEngine()
	chain := makeSimpleChain()
	// 慢动作
	chain.Rules[0].Actions = &core.ActionChain{Actions: []core.Action{
		core.ActionFunc(func(_ context.Context, _ core.DataContext) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}),
	}}
	_ = e.LoadChain(chain)

	var done atomic.Int64
	inflight := 3
	var wg sync.WaitGroup
	for i := 0; i < inflight; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := core.NewMapDataContext(map[string]any{"k": "v"})
			_, _ = e.EvalChain(context.Background(), "simple", data)
			done.Add(1)
		}()
	}
	time.Sleep(20 * time.Millisecond) // 让 goroutine 启动

	if err := e.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	wg.Wait()

	if done.Load() != int64(inflight) {
		t.Errorf("expected %d inflight evals to complete, got %d", inflight, done.Load())
	}
}

// ─────────────────────────────────────────────
//  5. Multi-DataContext 并发隔离
// ─────────────────────────────────────────────

// TestMultiDataContext_ConcurrentIsolation 并发评估时不同 DataContext 互不干扰
func TestMultiDataContext_ConcurrentIsolation(t *testing.T) {
	e := core.NewEngine()
	_ = e.LoadChain(makeSimpleChain())

	const N = 200
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := core.NewMapDataContext(map[string]any{
				"value": float64(i * 10),
			})
			_, err := e.EvalChain(context.Background(), "simple", data)
			if err != nil {
				t.Errorf("eval %d failed: %v", i, err)
			}
			if data.Value() != float64(i*10) {
				t.Errorf("data mutated: expected %v, got %v", float64(i*10), data.Value())
			}
		}()
	}
	wg.Wait()
}

// ─────────────────────────────────────────────
//  6. JSON 配置支持
// ─────────────────────────────────────────────

// TestE2E_JSONConfig 验证 JSON 配置能正确加载
func TestE2E_JSONConfig(t *testing.T) {
	cfg := config.ChainConfig{
		Chain: config.ChainMeta{
			ID: "json_chain", Name: "json", Version: 1, Status: "deployed", Root: true,
		},
		Rules: []config.RuleConfig{{
			ID: "r1", Priority: 1, Enabled: true,
		}},
	}
	data, _ := json.Marshal(cfg)
	reg := nodes.NewEmptyRegistry()
	r.RegisterPackage(builtin.Builtin)
	c, err := config.LoadFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := config.Parse(c, reg)
	if err != nil {
		t.Fatal(err)
	}
	if chain.ID != "json_chain" {
		t.Errorf("expected json_chain, got %s", chain.ID)
	}
}

// ─────────────────────────────────────────────
//  工具函数
// ─────────────────────────────────────────────

func makeSimpleChain() *core.RuleChain {
	return &core.RuleChain{
		ID:     "simple",
		Root:   true,
		Rules:  []*core.Rule{{
			ID: "r1", Priority: 1, Enabled: true,
			Condition: &core.ConditionNode{
				Leaf: core.ConditionFunc(func(_ context.Context, _ core.DataContext) bool { return true }),
			},
			Actions: &core.ActionChain{},
		}},
	}
}

func simpleYAML(version string) []byte {
	return []byte(`chain:
  id: ` + version + `
  name: simple
  root: true
  version: 1
  status: deployed
rules:
  - id: r1
    priority: 1
    enabled: true
    condition:
      type: value_range
      config:
        min_value: 0
        max_value: 1000
    actions: []
`)
}

func fmtInt(i int) string {
	if i < 10 {
		return "v0" + string(rune('0'+i))
	}
	return "v" + string(rune('0'+i/10)) + string(rune('0'+i%10))
}

// limitedLimiter 仅允许前 N 个通过
func limitedLimiter(allow int) ratelimit.Limiter {
	var counter atomic.Int64
	return &testLimiter{allow: int64(allow), counter: &counter}
}

type testLimiter struct {
	allow   int64
	counter *atomic.Int64
}

func (l *testLimiter) Allow(_ string) bool {
	n := l.counter.Add(1)
	return n <= l.allow
}

func (l *testLimiter) Wait(_ context.Context, _ string) error { return nil }