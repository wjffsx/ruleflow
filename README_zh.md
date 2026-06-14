# RuleFlow

[![Go Reference](https://pkg.go.dev/badge/github.com/wjffsx/ruleflow.svg)](https://pkg.go.dev/github.com/wjffsx/ruleflow)
[![CI](https://github.com/wjffsx/ruleflow/actions/workflows/ci.yml/badge.svg)](https://github.com/wjffsx/ruleflow/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wjffsx/ruleflow)](https://goreportcard.com/report/github.com/wjffsx/ruleflow)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**RuleFlow** 是一个用 Go 编写的高性能、零分配 IoT 规则引擎。它将规则链编译为函数闭包，实现亚微秒级热路径求值；通过写时复制（Copy-on-Write）实现无锁热加载；并提供可插拔的合约层用于指标采集、链路追踪、日志、限流和背压控制。

[English](README.md) | 中文

---

## 特性

- **编译-执行分离** — 规则链被编译为预分配的函数闭包，运行时求值零堆分配
- **写时复制热加载** — 原子加载/卸载规则链，读侧无锁。文件监听器通过 fsnotify 支持 YAML/JSON 热加载
- **可插拔错误处理** — Continue、Abort、Retry、Fallback 四种策略，支持装饰器链式组合
- **四级背压控制** — Normal → Degraded → Paused → Dropping，负载过高时自动跳过低优先级规则
- **FastPath 分类** — 编译器在编译期对规则分类，快速规则（<200ns）跳过慢路径开销
- **可插拔合约层** — 零依赖核心接口：MetricsSink、Logger、Limiter、Tracer、Health，可接入任意可观测性方案
- **灵活的条件树** — AND/OR/NOT 组合，叶节点支持设备类型、测点名称（正则/Trie 前缀）、数值范围、质量码、时间窗口、状态变化、动态阈值
- **可扩展节点注册表** — 通过简单的工厂接口注册自定义 Condition 和 Action 实现
- **YAML/JSON 配置** — 声明式规则链定义，两阶段解析、校验和冲突检测
- **多输入聚合** — MultiDataContext 支持连接池、汇聚检测和超时清理
- **三级节点体系** — 内置 IoT 通用节点 + IoT 扩展节点（表达式语言、存储、聚合）+ VPP（虚拟电厂）领域节点

---

## 架构

```
                     ┌─────────────────────────────────────────┐
                     │              应用层                      │
                     └──────┬──────────┬──────────┬────────────┘
                            │          │          │
                     ┌──────▼──┐ ┌─────▼─────┐ ┌──▼──────────┐
                     │ Router  │ │  Config    │ │  Adapter    │
                     │(可选)    │ │(YAML+热    │ │(背压/DLQ)   │
                     │         │ │  加载)     │ │             │
                     └──────┬──┘ └─────┬─────┘ └──┬──────────┘
                            │          │          │
          ┌─────────────────▼──────────▼──────────▼──────────────┐
          │                    核心引擎                            │
          │  ┌───────────┐   ┌──────────┐  ┌──────────────────┐  │
          │  │ Compiler  │   │  Engine  │  │  ErrorHandler    │  │
          │  │(闭包编译) │   │(COW+求值)│  │(Continue/Abort/  │  │
          │  │           │   │          │  │ Retry/Fallback)  │  │
          │  └───────────┘   └──────────┘  └──────────────────┘  │
          │  ┌──────────────────────────────────────────────────┐ │
          │  │              合约层                               │ │
          │  │  MetricsSink / Logger / Limiter / Tracer /       │ │
          │  │  Indicator / Tracker / Health / ShutdownState    │ │
          │  └──────────────────────────────────────────────────┘ │
          └────────────────────────┬─────────────────────────────┘
                                   │
                     ┌─────────────▼──────────────┐
                     │      节点注册表              │
                     │  ConditionFactory /         │
                     │  ActionFactory /            │
                     │  NodePackage                │
                     └──┬──────────┬──────────┬───┘
                        │          │          │
                 ┌──────▼──┐ ┌────▼─────┐ ┌──▼──────────┐
                 │ Builtin │ │   Ext    │ │ Extensions  │
                 │(IoT通用)│ │(IoT扩展) │ │(VPP领域)    │
                 │ 无依赖  │ │ 依赖注入 │ │ 电力/能源   │
                 └─────────┘ └──────────┘ └─────────────┘

                     ┌─────────────────────────────────────────┐
                     │              Contrib                      │
                     │  Prometheus / MemorySink / slog / otel    │
                     │  TokenBucket / MemoryState / Profiler    │
                     │  pprof / Debug EventBus / CircuitBreaker │
                     └─────────────────────────────────────────┘
```

### 包结构

```
pkg/ruleflow/
├── core/           # 引擎核心：编译器、求值器、类型、合约
│   ├── compiler/   # 规则链编译器（闭包预编译）
│   └── contract/   # 零依赖接口（MetricsSink、Logger 等）
│   └── engine/     # 求值引擎，COW 热加载
├── nodes/          # 注册表：ConditionFactory、ActionFactory、NodePackage
├── builtin/        # 内置 IoT 通用条件 & 动作节点
│   ├── condition/  # DeviceType、PointName、ValueRange、TimeWindow 等
│   └── action/     # Transform、Rename、Tag、Drop、Route、LimitCheck、Delay
├── ext/            # 需要依赖注入的扩展节点
│   ├── condition/  # ExprFilter、HistoricalCompare
│   └── action/     # StorageWrite、AggregationWrite、CalcNode 等
├── extensions/     # VPP（虚拟电厂）领域专用节点
│   ├── condition/  # SOC、PowerFactor、Frequency、RampRate 等
│   ├── action/     # Aggregator、DispatchControl、MarketPrice、CarbonCalc
│   └── flow/       # MsgGenerator、SubChain
├── config/         # YAML/JSON 配置加载器、校验器、文件监听器
├── datacontext/    # MapDataContext、MultiDataContext、MultiInputBuffer
├── router/         # 可选数据路由（pipelineType + 输入索引）
├── adapter/        # 外部系统适配器（背压、DLQ）
├── debug/          # 调试事件总线，用于规则求值追踪
└── contrib/        # 可选集成
    ├── prometheus/  # Prometheus MetricsSink
    ├── otel/        # OpenTelemetry TracerProvider
    ├── slog/        # log/slog Logger 适配器
    ├── tokenbucket/ # 内存令牌桶限流器
    └── circuitbreaker/ # 熔断器
```

---

## 快速开始

```go
package main

import (
    "context"
    "fmt"

    "github.com/wjffsx/ruleflow/pkg/ruleflow/core"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/action"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/datacontext"
)

func main() {
    eng := engine.NewEngine()

    // 编程式构建规则链
    chain := &core.RuleChain{
        ID: "demo", Name: "Demo Chain", Root: true, Version: 1, Status: "deployed",
        Rules: []*core.Rule{
            {
                ID: "rule_1", Priority: 1, Enabled: true,
                Condition: &core.ConditionNode{
                    Leaf: condition.NewDeviceTypeCondition("c1", []string{"analog"}),
                },
                Actions: &core.ActionChain{
                    Actions: []core.Action{
                        action.NewTransformAction("a1", &scale, nil, "kV"),
                    },
                },
                Targets: []string{"default"},
            },
        },
    }

    eng.LoadChain(chain)

    data := datacontext.NewMapDataContext(map[string]any{
        "device_id":  "sensor-01",
        "point_name": "voltage",
        "value":     220.5,
        "quality":   192,
    })

    result, err := eng.EvalChain(context.Background(), "demo", data)
    if err != nil {
        panic(err)
    }
    fmt.Printf("matched: %v, dropped: %v\n", result.Matched, result.Dropped)
}
```

### YAML 配置

```yaml
# chain.yaml
chain:
  id: "demo_chain"
  name: "Demo Chain"
  version: 1
  status: "deployed"
  pipeline_type: "analog"
  inputs:
    - point_name: "voltage"
      point_type: "analog"
  rules:
    - id: "rule_1"
      priority: 1
      condition:
        leaf:
          type: "device_id"
          config:
            values: ["sensor-01", "sensor-02"]
      actions:
        - type: "transform"
          config:
            scale: 1000
            unit: "mV"
        - type: "limit_check"
          config:
            upper_limit: 250000
            lower_limit: 0
```

加载方式：

```go
import "github.com/wjffsx/ruleflow/pkg/ruleflow/config"

// 文件监听模式（热加载）
watcher := config.NewFileWatcher("chain.yaml", loader)
watcher.Start()
defer watcher.Stop()
```

---

## 示例

| 示例 | 说明 |
|------|------|
| [basic](examples/basic/) | 最简示例：引擎创建、规则链构建、求值 |
| [custom-components](examples/custom-components/) | 自定义 Condition/Action 实现，使用 MapDataContext |
| [hot-reload](examples/hot-reload/) | 基于 fsnotify 的文件监听热加载 |
| [iot-gateway](examples/iot-gateway/) | IoT 网关场景：设备过滤、变换、路由 |
| [multi-tenant](examples/multi-tenant/) | 多租户规则隔离，每个租户独立引擎 |
| [observability-grpc-debug](examples/observability-grpc-debug/) | gRPC 调试端点，用于规则求值追踪 |
| [observability-grpc-health](examples/observability-grpc-health/) | gRPC 健康检查集成 |
| [observability-http](examples/observability-http/) | HTTP 可观测性端点（指标、pprof、健康检查） |

---

## 性能

- **快速规则**：热路径单规则求值 <200ns（零堆分配）
- **慢速规则**：单规则求值 <5µs（涉及正则、外部调用）
- **FastPath 分类**：编译器在编译期对规则分类，快速规则完全跳过慢路径开销
- **无反射**：所有节点配置在编译期解析，热路径为纯函数调用
- **sync.Pool**：EvalResult、MultiDataContext 等临时对象使用对象池

```
BenchmarkEvalFastRule-16         10000000   185.2 ns/op       0 B/op    0 allocs/op
BenchmarkEvalSlowRule-16           500000    4123 ns/op      48 B/op    2 allocs/op
BenchmarkEvalChain-16             2000000     892 ns/op       0 B/op    0 allocs/op
```

---

## 内置节点

### 条件节点

| 类型 | 说明 |
|------|------|
| `device_type` | 按设备类型过滤（预编译 map，O(1)） |
| `device_id` | 按设备 ID 过滤（预编译 map，O(1)） |
| `point_name` | 按测点名称过滤（预编译 map，O(1)） |
| `point_name_pattern` | 正则匹配测点名称（预编译正则） |
| `fqn_prefix` | FQN 前缀匹配（Trie，O(k)） |
| `value_range` | 数值范围过滤 |
| `value_in` | 离散值集合匹配（预编译 map，O(1)） |
| `quality` | 质量码过滤 |
| `limit_exceeded` | 越限状态检查 |
| `time_window` | 时间窗口（跨午夜、星期、时区感知） |
| `state_change` | 检测数值变化（使用 PreviousValue） |
| `dynamic_threshold` | 从 DataContext 标签读取阈值 |

### 动作节点

| 类型 | 说明 |
|------|------|
| `transform` | 缩放 + 偏移 + 单位转换 |
| `rename` | 通过 `_rename` 标签重命名数据点 |
| `tag` | 添加键值对标签 |
| `drop` | 丢弃数据点（返回 ErrDropData） |
| `route` | 添加路由目标 |
| `limit_check` | 检测上下限越限 |
| `delay` | 异步延迟执行内嵌动作 |

---

## 条件树

条件支持 AND/OR/NOT 组合，实现复杂逻辑：

```yaml
# AND：所有子条件必须匹配
condition:
  operator: and
  children:
    - leaf:
        type: "device_type"
        config:
          values: ["analog"]
    - leaf:
        type: "value_range"
        config:
          min: 0
          max: 100

# OR：任一子条件匹配即可
condition:
  operator: or
  children:
    - leaf:
        type: "device_id"
        config:
          values: ["sensor-01"]
    - leaf:
        type: "device_id"
        config:
          values: ["sensor-02"]

# NOT：取反
condition:
  operator: not
  children:
    - leaf:
        type: "quality"
        config:
          min_quality: 192

# 嵌套：AND 嵌套在 OR 内
condition:
  operator: or
  children:
    - operator: and
      children:
        - leaf:
            type: "device_type"
            config:
              values: ["analog"]
        - leaf:
            type: "value_range"
            config:
              min: 10
              max: 50
    - leaf:
        type: "device_id"
        config:
          values: ["sensor-01"]
```

---

## 自定义节点

### 自定义条件节点

```go
import (
    "context"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// MaintenanceWindowCondition 仅在配置的时间窗口内返回 true
type MaintenanceWindowCondition struct {
    id        string
    startHour int
    endHour   int
}

func (c *MaintenanceWindowCondition) ID() string { return c.id }

func (c *MaintenanceWindowCondition) Evaluate(_ context.Context, data core.DataContext) bool {
    hour := data.GetTag("current_hour")
    // ... 解析 hour 并比较
    return true
}
```

### 自定义动作节点

```go
type WebhookAction struct {
    id  string
    url string
}

func (a *WebhookAction) ID() string { return a.id }

func (a *WebhookAction) Execute(_ context.Context, data core.DataContext) error {
    // 发送数据到外部 webhook
    // 返回 core.ErrDropData 可丢弃数据点并停止链求值
    return nil
}
```

### 注册方式

```go
import "github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"

reg := nodes.NewEmptyRegistry()

// 方式 A：注册单个工厂
reg.RegisterConditionFactory("maintenance_window", func(id string, config map[string]any) (core.Condition, error) {
    return &MaintenanceWindowCondition{
        id:        id,
        startHour: config["start_hour"].(int),
        endHour:   config["end_hour"].(int),
    }, nil
})

// 方式 B：实现 NodePackage 接口（推荐用于库）
reg.RegisterPackage(MyPackage{})
```

---

## 热加载机制

```
文件变更检测（fsnotify）
       │
       ▼
┌─────────────────────┐
│ 200ms 防抖          │ ← 防止频繁触发
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ 加载 YAML/JSON      │
│ 解析为中间结构       │ ← 第一阶段：仅字段映射
│ 校验                 │ ← 类型检查、引用检查
│ 冲突检测             │ ← 如：重叠条件
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ 使用注册表解析       │ ← 第二阶段：实例化条件/动作
│ 解析替换引用         │
│ 编译为闭包           │
│ 检测循环依赖         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ COW 交换            │ ← 构建新快照
│ snapshot.Store(new) │ ← 原子存储，现有读者不受影响
└─────────────────────┘
```

---

## 背压控制

```
四级背压：

  Normal    → 处理所有规则
  Degraded  → 跳过优先级低于阈值的规则
  Paused    → 跳过所有规则（接收数据但不处理）
  Dropping  → 递增丢弃计数器，立即返回
```

---

## 可观测性

RuleFlow 核心仅依赖零开销的接口定义，所有可观测性实现都是可选的：

| 合约接口 | 说明 | Contrib 实现 |
|----------|------|-------------|
| `MetricsSink` | 指标采集 | Prometheus |
| `Logger` | 结构化日志 | log/slog |
| `Tracer` | 链路追踪 | OpenTelemetry |
| `Limiter` | 限流 | TokenBucket |
| `Indicator` | 背压指示 | — |
| `Health` | 健康检查 | — |

未配置时自动使用 Noop 实现，引擎零开销运行。

---

## 参与贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md) 了解开发流程、代码规范和 PR 指南。

### 贡献者快速开始

```bash
git clone https://github.com/wjffsx/ruleflow.git
cd ruleflow
go mod download
go test -count=1 -race ./pkg/...
```

---

## 许可证

RuleFlow 基于 [Apache License, Version 2.0](LICENSE) 许可。
