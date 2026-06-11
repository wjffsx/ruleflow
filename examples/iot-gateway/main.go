// Package main 演示 IoT 网关典型场景：从配置文件加载规则链，评估数据点。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vpptu/ruleflow/pkg/ruleflow/builtin"
	"github.com/vpptu/ruleflow/pkg/ruleflow/config"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/engine"
	"github.com/vpptu/ruleflow/pkg/ruleflow/datacontext"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes"
)

// chainYAML 是典型 IoT 网关规则链定义
const chainYAML = `
chain:
  id: iot_gateway
  name: IoT Gateway Pipeline
  description: Filter, transform and route IoT data points
  root: true
  version: 1
  status: deployed
rules:
  - id: drop_low_quality
    priority: 1
    enabled: true
    condition:
      type: quality
      config:
        min_quality: 192
    actions:
      - type: drop
        config: {}

  - id: transform_temperature
    priority: 10
    enabled: true
    condition:
      type: point_name_pattern
      config:
        pattern: ".*temperature.*"
    actions:
      - type: transform
        config:
          scale: 1.0
          offset: -273.15
          unit: "C"

  - id: alarm_high_temp
    priority: 20
    enabled: true
    condition:
      type: value_range
      config:
        min_value: 80.0
        max_value: 1000.0
    actions:
      - type: alarm_notify
        config:
          severity: critical
          message: "high temperature alarm"
`

func main() {
	// 1. 写入临时 YAML
	dir, _ := os.MkdirTemp("", "iot-gw-")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "chain.yaml")
	if err := os.WriteFile(path, []byte(chainYAML), 0644); err != nil {
		log.Fatal(err)
	}

	// 2. 加载并解析
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	reg := nodes.NewEmptyRegistry()
	reg.RegisterPackage(builtin.Builtin)
	chain, err := config.Parse(cfg, reg)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}

	// 3. 编译到引擎
	e := engine.NewEngine()
	if err := e.LoadChain(chain); err != nil {
		log.Fatalf("load chain: %v", err)
	}

	// 4. 模拟 3 个不同类型的数据点
	samples := []map[string]any{
		{"device_id": "sensor-1", "point_name": "temperature", "value": 25.0, "quality": 192},
		{"device_id": "sensor-2", "point_name": "humidity", "value": 60.0, "quality": 100},     // 质量差，应被 drop
		{"device_id": "sensor-3", "point_name": "temperature", "value": 350.0, "quality": 192}, // 触发高温告警
	}

	for _, s := range samples {
		data := datacontext.NewMapDataContext(s)
		result, _ := e.EvalChain(context.Background(), "iot_gateway", data)
		status := "ok"
		if data.Dropped() {
			status = "DROPPED"
		}
		fmt.Printf("[%s] device=%s point=%s value=%v -> matched=%d dropped=%v\n",
			status, s["device_id"], s["point_name"], s["value"],
			len(result.MatchedRules), data.Dropped())
	}

	// 5. 健康检查 + 优雅关闭
	hc := e.HealthCheck()
	fmt.Printf("\nEngine health: %s, chains=%d, uptime=%ds\n",
		hc.Status, hc.LoadedChains, hc.UptimeSeconds)

	if err := e.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
