// Package main 演示热重载：基于 fsnotify 监听 YAML 文件，修改后自动应用。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/config"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/datacontext"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
)

func main() {
	// 1. 准备临时配置文件
	dir, _ := os.MkdirTemp("", "hot-reload-")
	defer os.RemoveAll(dir)
	cfgPath := filepath.Join(dir, "chain.yaml")

	initialYAML := `chain:
  id: hr_chain
  name: hot-reload demo
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
    actions: []
`
	if err := os.WriteFile(cfgPath, []byte(initialYAML), 0644); err != nil {
		log.Fatal(err)
	}

	// 2. 启动文件监听器
	e := engine.NewEngine()
	reg := nodes.NewEmptyRegistry()
	reg.RegisterPackage(builtin.Builtin)
	fw, err := config.NewFileWatcher(cfgPath, &config.DefaultLoader{
		Registry: reg,
		OnLoad:   func(ctx context.Context, chain *core.RuleChain) error {
			e.UnloadChain(chain.ID)
			return e.LoadChain(chain)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fw.WithDebounce(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fw.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer fw.Stop()

	// 3. 持续评估
	go func() {
		for i := 0; i < 5; i++ {
			data := datacontext.NewMapDataContext(map[string]any{"value": float64(i * 10)})
			result, _ := e.EvalChain(ctx, "hr_chain", data)
			fmt.Printf("tick %d: matched=%d\n", i, len(result.MatchedRules))
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// 4. 等待评估启动后修改配置
	time.Sleep(300 * time.Millisecond)
	v2 := `chain:
  id: hr_chain_v2
  name: hot-reload demo v2
  root: true
  version: 2
  status: deployed
rules:
  - id: r1
    priority: 1
    enabled: true
    condition:
      type: value_range
      config:
        min_value: 0
        max_value: 50
    actions: []
`
	fmt.Println("[demo] rewriting config to v2...")
	if err := os.WriteFile(cfgPath, []byte(v2), 0644); err != nil {
		log.Fatal(err)
	}

	// 5. 再次评估 — 应能命中 v2 的链
	time.Sleep(500 * time.Millisecond)
	data := datacontext.NewMapDataContext(map[string]any{"value": 25.0})
	result, _ := e.EvalChain(ctx, "hr_chain_v2", data)
	fmt.Printf("v2 eval: matched=%d\n", len(result.MatchedRules))
}
