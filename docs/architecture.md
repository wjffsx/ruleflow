# RuleFlow Architecture

## Design Philosophy

RuleFlow is designed around three core principles:

1. **Compile-Execute separation** — Rule chain configuration is expensive; runtime evaluation must be cheap. The compiler pre-processes as much work as possible (closure generation, map compilation, regex compilation, trie construction) so the hot path is pure function calls with zero heap allocation.

2. **Contract-Implementation separation** — The core engine depends only on small, hand-written interfaces in the `contract` package (MetricsSink, Logger, Limiter, Tracer, etc.). Concrete implementations live in `contrib/` and are selected by the application. This keeps the core dependency-free and testable.

3. **Copy-on-write state management** — Hot-reload must not block readers. Rule chain snapshots are swapped atomically via `atomic.Pointer`. Readers always see a consistent snapshot; writers build a new snapshot and swap in one instruction.

---

## Package Dependency Graph

```
                          ┌──────────────┐
                          │  Application │
                          └──┬───────┬───┘
                             │       │
                    ┌────────▼──┐ ┌──▼──────────┐
                    │ config/   │ │ engine/      │
                    │ (YAML)    │ │ (COW+eval)   │
                    └────────┬──┘ └──┬───────────┘
                             │       │
                    ┌────────▼───────▼───────────┐
                    │         core/               │
                    │  ┌─────────┐ ┌───────────┐ │
                    │  │compiler/│ │ contract/  │ │
                    │  └─────────┘ └───────────┘ │
                    │  ┌─────────┐ ┌───────────┐ │
                    │  │types.go │ │interfaces │ │
                    │  │         │ │  .go      │ │
                    │  └─────────┘ └───────────┘ │
                    └───────────────┬────────────┘
                                    │
                    ┌───────────────▼────────────┐
                    │         nodes/              │
                    │   NewEmptyRegistry()        │
                    │   RegisterPackage()         │
                    │   CreateCondition/Action    │
                    └──┬──────────┬──────────┬───┘
                       │          │          │
                ┌──────▼──┐ ┌────▼─────┐ ┌──▼──────────┐
                │ builtin/│ │   ext/   │ │ extensions/ │
                │(IoT-gen)│ │(IoT-ext) │ │(VPP domain) │
                └─────────┘ └──────────┘ └─────────────┘
```

### Dependency rules

- `core/` imports nothing outside itself and the standard library.
- `nodes/` imports `core/` for the `Condition`/`Action` interfaces.
- `builtin/`, `ext/`, `extensions/` import `nodes/` for the `NodePackage` interface.
- `config/` imports `core/` and `nodes/` for two-phase parsing.
- `engine/` imports `core/` and optionally `nodes/` for runtime.
- `contrib/*` implements `core/contract` interfaces only.

---

## Core Engine Architecture

### Compilation Pipeline

```
RuleChain (config)          CompiledChain (runtime)
       │                           │
       │  compiler.CompileChain()  │
       │                           │
       ▼                           ▼
┌──────────────┐          ┌───────────────────┐
│  Compiler    │          │ CompiledChain      │
│              │          │                    │
│ 1. Validate  │ ──────►  │ ├ ID/Version      │
│ 2. Build     │          │ ├ FastRules []     │
│    dep graph │          │ ├ SlowRules []     │
│ 3. Detect    │          │ ├ PrewarmFunc      │
│    cycles    │          │ └ SnapshotTime      │
│ 4. Pre-      │          │                    │
│    compile   │          │ ┌──────────────┐   │
│    closures  │          │ │ CompiledRule  │   │
│ 5. Classify  │          │ │              │   │
│    FastPath  │          │ │ ├ EvaluateFunc│   │
└──────────────┘          │ │ ├ ExecuteFunc │   │
                          │ │ └ IsFastPath  │   │
                          │ └──────────────┘   │
                          └───────────────────┘
```

### Condition tree compilation

Condition trees are compiled into a single closure:

```go
// Before compilation (interpreted tree):
root := &ConditionNode{
    Operator: AND,
    Children: []*ConditionNode{
        {Leaf: deviceTypeCond},
        {Leaf: valueRangeCond},
    },
}

// After compilation (single closure):
evaluateFunc := func(ctx context.Context, data core.DataContext) bool {
    return deviceTypeCond.Evaluate(ctx, data) && valueRangeCond.Evaluate(ctx, data)
}
```

### Action chain compilation

Action chains are compiled into a single closure that iterates pre-checked actions:

```go
executeFunc := func(ctx context.Context, data core.DataContext) error {
    for _, action := range actions {
        if err := action.Execute(ctx, data); err != nil {
            if errors.Is(err, core.ErrDropData) {
                return err
            }
            // delegate to error handler
        }
    }
    return nil
}
```

### FastPath classification

A rule qualifies for FastPath when ALL of the following are true:

- All conditions are stateless (no `StateStore` dependency)
- All actions are stateless (no external IO)
- No side effects outside the DataContext

Fast rules bypass error handler dispatching, metrics collection (except counters), and tracing. The compiler sets `IsFastPath` at compile time.

---

## Engine Evaluation Flow

```
EvalChain(ctx, chainID, data)
       │
       ▼
┌──────────────────┐
│ 1. Shutdown check│ ← if ShuttingDown, return ErrEngineShutdown
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ 2. Backpressure  │ ← consult Indicator
│    level check   │    Normal → proceed
│                  │    Degraded → skip low-priority
│                  │    Paused → skip all
│                  │    Dropping → increment drop counter, return
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ 3. COW snapshot  │ ← snapshot.Load() — zero lock
│    read          │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ 4. Trace/Timeout │ ← optional OpenTelemetry span
│    setup         │    optional context timeout from config
└──────┬───────────┘
       │
       ▼
┌────────────────────────────────────────────┐
│ 5. Rule evaluation loop                     │
│                                             │
│  for each CompiledRule (sorted by priority): │
│    if limiter.Allow(ruleID):                 │
│      if rule.EvaluateFunc(ctx, data):        │
│        err := rule.ExecuteFunc(ctx, data)    │
│        if err == ErrDropData:                │
│          return dropped                      │
│        consultErrorHandler(err)              │
│    if EvalModeFirst && matched:              │
│      break                                   │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌──────────────────┐
│ 6. Pool release  │ ← return EvalResult to sync.Pool
└──────────────────┘
```

---

## Error Handling Architecture

```
ErrorHandler interface:
    Handle(ctx, RuleFlowError) ErrorAction

Implementations:
    ContinueOnErrorHandler → returns Continue
    AbortOnErrorHandler   → returns Abort
    RetryOnceErrorHandler → retries once, then returns Fallback

ChainedHandler:
    Combines multiple handlers via decorator.
    E.g., RetryOnceErrorHandler → MetricsErrorHandler → ContinueOnErrorHandler
```

Error actions:

| Action | Behavior |
|--------|----------|
| Continue | Log + increment error counter, proceed to next rule |
| Abort | Terminate chain evaluation, return error immediately |
| Retry | Re-execute the action (with backoff if configured) |
| Fallback | Execute a fallback action, then continue |

---

## Hot-Reload Mechanism

```
File change detected (fsnotify)
       │
       ▼
┌─────────────────────┐
│ 200ms debounce      │ ← prevents thrashing
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Load YAML/JSON      │
│ Parse to Intermediate│ ← phase 1: field mapping only
│ Validate             │ ← type checks, reference checks
│ Detect conflicts     │ ← e.g., overlapping conditions
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Parse with Registry │ ← phase 2: instantiate conditions/actions
│ Resolve replaces    │
│ Compile to closures │
│ Detect cycles       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ COW swap            │ ← build new snapshot
│ snapshot.Store(new) │ ← atomic store, existing readers unaffected
└─────────────────────┘
```

---

## Backpressure Architecture

```
Indicator interface:
    Level() contract.Level  // Normal / Degraded / Paused / Dropping

Engine behavior per level:

  Normal    → process all rules
  Degraded  → skip rules with priority < configured threshold
  Paused    → skip all rules (accept data, don't process)
  Dropping  → increment drop counter, return immediately
```

The `adapter/backpressure.go` maps an application-level `BackpressureManager` to the engine's `contract.Indicator` interface.

---

## Observability Architecture

```
MetricsSink interface:
    IncrementEvalCount(chainID string)
    RecordEvalLatency(chainID string, dur time.Duration)
    IncrementConditionMatch(ruleID string, conditionID string)
    IncrementConditionMiss(ruleID string, conditionID string)
    IncrementActionSuccess(ruleID string, actionID string)
    IncrementActionFailure(ruleID string, actionID string)
    IncrementDropped(chainID string)
    RecordActionLatency(ruleID string, actionID string, dur time.Duration)

Logger interface:
    Debug(ctx, msg, keysAndValues...)
    Info(ctx, msg, keysAndValues...)
    Warn(ctx, msg, keysAndValues...)
    Error(ctx, msg, keysAndValues...)

Tracer interface:
    StartSpan(ctx, name) (context.Context, Span)
    (Span) End()
    (Span) SetAttributes(...)
```

All three interfaces have `Noop` implementations in the contract package, so
the engine works without any observability setup. Applications enable
observability by passing non-noop implementations through engine options.

---

## DataContext Design

### Zero-allocation contract

The `DataContext` interface is designed to avoid heap allocation on the hot path:

```go
// BAD — returns slice, forces allocation
Tags() map[string]string
Targets() []string

// GOOD — accessor methods, no allocation
GetTag(key string) string
TargetCount() int
TargetAt(i int) string
```

### Pooling

- `EvalResult` is pooled via `sync.Pool`.
- `MultiDataContext` is pooled via `MultiDataContextPool`.
- `MultiInputBuffer` entries are cleaned up after a configurable timeout (default 5s).

### Common implementations

| Implementation | Use case |
|----------------|----------|
| `MapDataContext` | General-purpose, thread-safe (sync.RWMutex), built from `map[string]any` |
| `MultiDataContext` | Multi-input aggregation, gather detection |
| `simpleDataPoint` | Example code (see examples/basic) |
