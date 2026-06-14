# RuleFlow

[![Go Reference](https://pkg.go.dev/badge/github.com/wjffsx/ruleflow.svg)](https://pkg.go.dev/github.com/wjffsx/ruleflow)
[![CI](https://github.com/wjffsx/ruleflow/actions/workflows/ci.yml/badge.svg)](https://github.com/wjffsx/ruleflow/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wjffsx/ruleflow)](https://goreportcard.com/report/github.com/wjffsx/ruleflow)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**RuleFlow** is a high-performance, zero-allocation IoT rule engine written in Go. It compiles rule chains into function closures for sub-microsecond evaluation on the hot path, supports lock-free hot-reload via copy-on-write, and provides a pluggable contract layer for metrics, tracing, logging, rate limiting, and backpressure.

English | [中文](README_zh.md)

---

## Features

- **Compile-Execute separation** — Rule chains are compiled into pre-allocated function closures; runtime evaluation incurs zero heap allocation.
- **Copy-on-write hot-reload** — Load and unload rule chains atomically with no read-side locking. File watcher supports YAML/JSON hot-reload via fsnotify.
- **Pluggable error handling** — Continue, Abort, Retry, and Fallback strategies with decorator chaining.
- **Four-level backpressure** — Normal → Degraded → Paused → Dropping. Skips low-priority rules automatically under load.
- **FastPath classification** — The compiler classifies rules at compile time. Fast rules (<200ns) bypass slow-path bookkeeping.
- **Pluggable contract layer** — Zero-dependency core interfaces for MetricsSink, Logger, Limiter, Tracer, and Health. Bring your own observability stack.
- **Expressive condition trees** — AND/OR/NOT composition with leaf nodes for device type, point name (regex/trie prefix), value range, quality, time window, state change, and dynamic thresholds.
- **Extensible node registry** — Register custom Condition and Action implementations via a simple factory interface.
- **YAML/JSON configuration** — Declarative rule chain definition with two-phase parsing, validation, and conflict detection.
- **Multi-input aggregation** — MultiDataContext with pooling, gather detection, and timeout-based cleanup.
- **Builtin + Ext + Domain-specific nodes** — Ship with IoT-generic nodes, IoT-extension nodes (expr-lang, storage, aggregation), and VPP (virtual power plant) domain nodes.

---

## Architecture

```
                     ┌─────────────────────────────────────────┐
                     │           Application Layer             │
                     └──────┬──────────┬──────────┬────────────┘
                            │          │          │
                     ┌──────▼──┐ ┌─────▼─────┐ ┌──▼──────────┐
                     │ Router  │ │  Config    │ │  Adapter    │
                     │(optional)│ │(YAML+hot-  │ │(backpressure│
                     │         │ │  reload)   │ │  /DLQ)     │
                     └──────┬──┘ └─────┬─────┘ └──┬──────────┘
                            │          │          │
          ┌─────────────────▼──────────▼──────────▼──────────────┐
          │                    Core Engine                        │
          │  ┌───────────┐   ┌──────────┐  ┌──────────────────┐  │
          │  │ Compiler  │   │  Engine  │  │  ErrorHandler    │  │
          │  │(closures) │   │(COW+eval)│  │(Continue/Abort/  │  │
          │  │           │   │          │  │ Retry/Fallback)  │  │
          │  └───────────┘   └──────────┘  └──────────────────┘  │
          │  ┌──────────────────────────────────────────────────┐ │
          │  │              Contract Layer                       │ │
          │  │  MetricsSink / Logger / Limiter / Tracer /       │ │
          │  │  Indicator / Tracker / Health / ShutdownState    │ │
          │  └──────────────────────────────────────────────────┘ │
          └────────────────────────┬─────────────────────────────┘
                                   │
                     ┌─────────────▼──────────────┐
                     │      Nodes Registry        │
                     │  ConditionFactory /         │
                     │  ActionFactory /            │
                     │  NodePackage                │
                     └──┬──────────┬──────────┬───┘
                        │          │          │
                 ┌──────▼──┐ ┌────▼─────┐ ┌──▼──────────┐
                 │ Builtin │ │   Ext    │ │ Extensions  │
                 │(IoT-gen)│ │(IoT-ext) │ │(VPP domain) │
                 │ no deps │ │ injected │ │ power/energy│
                 └─────────┘ └──────────┘ └─────────────┘

                     ┌─────────────────────────────────────────┐
                     │              Contrib                      │
                     │  Prometheus / MemorySink / slog / otel    │
                     │  TokenBucket / MemoryState / Profiler    │
                     │  pprof / Debug EventBus / CircuitBreaker │
                     └─────────────────────────────────────────┘
```

### Package layout

```
pkg/ruleflow/
├── core/           # Engine core: compiler, evaluator, types, contracts
│   ├── compiler/   # Rule chain compiler (closure pre-compilation)
│   └── contract/   # Zero-dependency interfaces (MetricsSink, Logger, etc.)
│   └── engine/     # Evaluation engine with COW hot-reload
├── nodes/          # Registry: ConditionFactory, ActionFactory, NodePackage
├── builtin/        # Builtin IoT-generic condition & action nodes
│   ├── condition/  # DeviceType, PointName, ValueRange, TimeWindow, etc.
│   └── action/     # Transform, Rename, Tag, Drop, Route, LimitCheck, Delay
├── ext/            # Extension nodes requiring dependency injection
│   ├── condition/  # ExprFilter, HistoricalCompare
│   └── action/     # StorageWrite, AggregationWrite, CalcNode, etc.
├── extensions/     # VPP (Virtual Power Plant) domain-specific nodes
│   ├── condition/  # SOC, PowerFactor, Frequency, RampRate, etc.
│   ├── action/     # Aggregator, DispatchControl, MarketPrice, CarbonCalc
│   └── flow/       # MsgGenerator, SubChain
├── config/         # YAML/JSON config loader, validator, file watcher
├── datacontext/    # MapDataContext, MultiDataContext, MultiInputBuffer
├── router/         # Optional data routing (pipelineType + input index)
├── adapter/        # External system adapters (backpressure, DLQ)
├── debug/          # Debug event bus for rule evaluation tracing
└── contrib/        # Optional integrations
    ├── prometheus/  # Prometheus MetricsSink
    ├── otel/        # OpenTelemetry TracerProvider
    ├── slog/        # log/slog Logger adapter
    ├── tokenbucket/ # In-memory token bucket rate limiter
    └── circuitbreaker/ # Circuit breaker
```

---

## Quick Start

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

    // Build a rule chain programmatically
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

### YAML configuration

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

Then load it:

```go
import "github.com/wjffsx/ruleflow/pkg/ruleflow/config"

// file watch mode
watcher := config.NewFileWatcher("chain.yaml", loader)
watcher.Start()
defer watcher.Stop()
```

---

## Examples

| Example | Description |
|---------|-------------|
| [basic](examples/basic/) | Minimal setup: engine creation, rule chain construction, evaluation |
| [custom-components](examples/custom-components/) | Custom Condition/Action implementations with MapDataContext |
| [hot-reload](examples/hot-reload/) | File watcher hot-reload with fsnotify |
| [iot-gateway](examples/iot-gateway/) | IoT gateway scenario: device filtering, transform, route |
| [multi-tenant](examples/multi-tenant/) | Multi-tenant rule isolation with per-tenant engines |
| [observability-grpc-debug](examples/observability-grpc-debug/) | gRPC debug endpoint for rule evaluation tracing |
| [observability-grpc-health](examples/observability-grpc-health/) | gRPC health check integration |
| [observability-http](examples/observability-http/) | HTTP observability endpoints (metrics, pprof, health) |

---

## Performance

- **Fast rules**: <200ns per rule evaluation on the hot path (zero heap allocation).
- **Slow rules**: <5µs per rule evaluation (conditions involving regex, external calls).
- **FastPath classification**: The compiler classifies rules at compile time. Fast rules skip slow-path bookkeeping entirely.
- **No reflection**: All node configurations are parsed at compile time; the hot path is pure function calls.
- **sync.Pool**: EvalResult, MultiDataContext, and other transient objects are pooled.

```
BenchmarkEvalFastRule-16         10000000   185.2 ns/op       0 B/op    0 allocs/op
BenchmarkEvalSlowRule-16           500000    4123 ns/op      48 B/op    2 allocs/op
BenchmarkEvalChain-16             2000000     892 ns/op       0 B/op    0 allocs/op
```

---

## Builtin Nodes

### Conditions

| Type | Description |
|------|-------------|
| `device_type` | Filter by device type (pre-compiled map, O(1)) |
| `device_id` | Filter by device ID (pre-compiled map, O(1)) |
| `point_name` | Filter by point name (pre-compiled map, O(1)) |
| `point_name_pattern` | Regex point name matching (pre-compiled regexp) |
| `fqn_prefix` | FQN prefix matching (Trie, O(k)) |
| `value_range` | Numeric range filter |
| `value_in` | Discrete value set matching (pre-compiled map, O(1)) |
| `quality` | Quality code filter |
| `limit_exceeded` | Limit violation state check |
| `time_window` | Time window (cross-midnight, day-of-week, timezone-aware) |
| `state_change` | Detect value transitions (uses PreviousValue) |
| `dynamic_threshold` | Read thresholds from DataContext tags |

### Actions

| Type | Description |
|------|-------------|
| `transform` | Scale + offset + unit conversion |
| `rename` | Rename data point via `_rename` tag |
| `tag` | Add key-value tags |
| `drop` | Drop data point (returns ErrDropData) |
| `route` | Add routing targets |
| `limit_check` | Detect upper/lower limit violations |
| `delay` | Async delayed execution of embedded action |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, code conventions, and pull request guidelines.

### Quick start for contributors

```bash
git clone https://github.com/wjffsx/ruleflow.git
cd ruleflow
go mod download
go test -count=1 -race ./pkg/...
```

---

## License

RuleFlow is licensed under the [Apache License, Version 2.0](LICENSE).
