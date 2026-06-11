# Custom Nodes Guide

RuleFlow's node system is extensible. You can register custom Condition and Action implementations to inject application-specific domain logic.

---

## Quick Overview

```
Condition interface:
    ID() string
    Evaluate(ctx context.Context, data core.DataContext) bool

Action interface:
    ID() string
    Execute(ctx context.Context, data core.DataContext) error

NodePackage interface:
    GetConditionFactories() map[string]nodes.ConditionFactory
    GetActionFactories() map[string]nodes.ActionFactory

Registry:
    NewEmptyRegistry()
    RegisterPackage(pkg NodePackage)
    CreateCondition(typeName, id, config)
    CreateAction(typeName, id, config)
```

---

## Custom Condition

```go
import (
    "context"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// MaintenanceWindowCondition evaluates to true only within a configured time window.
type MaintenanceWindowCondition struct {
    id          string
    startHour   int
    endHour     int
}

func (c *MaintenanceWindowCondition) ID() string { return c.id }

func (c *MaintenanceWindowCondition) Evaluate(_ context.Context, data core.DataContext) bool {
    // DataContext does not carry wall-clock time — the condition
    // inspects application state or uses the context's deadline.
    // Here we read a "current_hour" tag from the data point.
    hour := data.GetTag("current_hour")
    // ... parse hour and compare
    return true
}
```

## Custom Action

```go
type WebhookAction struct {
    id   string
    url  string
}

func (a *WebhookAction) ID() string { return a.id }

func (a *WebhookAction) Execute(_ context.Context, data core.DataContext) error {
    // Send data to external webhook
    // Return core.ErrDropData to discard the data point and stop chain evaluation
    return nil
}
```

---

## Registration

### Option A: Register individual factories

```go
import "github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"

reg := nodes.NewEmptyRegistry()

reg.RegisterConditionFactory("maintenance_window", func(id string, config map[string]any) (core.Condition, error) {
    return &MaintenanceWindowCondition{
        id:        id,
        startHour: config["start_hour"].(int),
        endHour:   config["end_hour"].(int),
    }, nil
})

reg.RegisterActionFactory("webhook", func(id string, config map[string]any) (core.Action, error) {
    return &WebhookAction{
        id:  id,
        url: config["url"].(string),
    }, nil
})
```

### Option B: Implement NodePackage (recommended for libraries)

```go
type MyPackage struct{}

func (MyPackage) GetConditionFactories() map[string]nodes.ConditionFactory {
    return map[string]nodes.ConditionFactory{
        "maintenance_window": func(id string, config map[string]any) (core.Condition, error) {
            return &MaintenanceWindowCondition{...}, nil
        },
    }
}

func (MyPackage) GetActionFactories() map[string]nodes.ActionFactory {
    return map[string]nodes.ActionFactory{
        "webhook": func(id string, config map[string]any) (core.Action, error) {
            return &WebhookAction{...}, nil
        },
    }
}

// Usage
reg.RegisterPackage(MyPackage{})
```

---

## Injecting the Registry into the Engine

```go
import (
    "github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/builtin"
)

reg := nodes.NewEmptyRegistry()
builtin.RegisterAll(reg)
reg.RegisterPackage(MyPackage{})

eng := engine.NewEngine(engine.WithRegistry(reg))
```

---

## Stateful Conditions

If your condition needs state across evaluations (e.g., state change detection, rate limiting), implement the `StatefulCondition` pattern:

```go
type MyStatefulCondition struct {
    id          string
    store       core.StateStore  // injected by the engine
    windowSize  time.Duration
}

func (c *MyStatefulCondition) ID() string { return c.id }

func (c *MyStatefulCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
    key := data.DeviceID() + "/" + data.PointName()
    if history, ok := c.store.Get(key); ok {
        // Use stored state
    }
    c.store.Set(key, time.Now())
    return true
}
```

The engine automatically wraps DataContext with `StatefulDataContext` when a StateStore is configured via `engine.WithStateStore()`.

---

## Prewarming

If your node needs to warm up caches or connection pools before the engine starts processing, implement the `Prewarmable` interface:

```go
type MyNode struct {
    client *ExternalClient
}

func (n *MyNode) Prewarm(ctx context.Context) error {
    var err error
    n.client, err = connectToExternal(ctx)
    return err
}
```

The engine calls `Prewarm()` on all registered nodes during `engine.Prewarm()`.

---

## Best Practices

1. **Keep the hot path allocation-free** — Pre-compute everything in the factory function (`New*`), not in `Evaluate`/`Execute`.
2. **Use `ErrDropData` for discard** — Return `core.ErrDropData` from an action to stop chain evaluation and signal that the data point was consumed.
3. **Config validation in factory** — Validate config in the factory function and return an error early, rather than panicking at evaluation time.
4. **Thread safety** — The engine guarantees that each DataContext is accessed by a single goroutine, but your node may be called concurrently with different DataContexts. Use `StateStore` for per-key state rather than mutex-protected maps.
5. **Document config fields** — Use `ComponentMeta` (in `nodes/meta.go`) to describe config fields for tooling and documentation generation.
