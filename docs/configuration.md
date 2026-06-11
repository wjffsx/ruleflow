# RuleFlow Configuration Guide

Rule chains can be defined declaratively in YAML (or JSON) and loaded at runtime. The configuration system supports two-phase parsing, validation, conflict detection, and hot-reload.

---

## Chain Configuration Format

```yaml
chain:
  # Required: unique chain identifier
  id: "my_chain"

  # Optional: human-readable name
  name: "My Rule Chain"

  # Optional: semantic version (default: 1)
  version: 1

  # Optional: deployment status (default: "deployed")
  # Values: "deployed", "draft", "disabled"
  status: "deployed"

  # Optional: whether this is a root chain (default: true)
  root: true

  # Optional: pipeline type for router integration
  # Values: "analog", "digital", "meter"
  pipeline_type: "analog"

  # Optional: input declarations (used by router for data routing)
  inputs:
    - point_name: "voltage"
      point_type: "analog"
      data_type: "float64"
      unit: "V"
    - point_name: "current"
      point_type: "analog"
      data_type: "float64"
      unit: "A"

  # Required: list of rules
  rules:
    - # ... see below
```

---

## Rule Configuration

```yaml
rules:
  - id: "rule_1"                    # Required: unique rule ID within this chain
    name: "Filter analog + transform"  # Optional
    priority: 1                     # Optional: lower = higher priority (default: 1)
    weight: 100                     # Optional: tie-breaker when priorities equal (default: 100)
    enabled: true                   # Optional (default: true)
    description: "..."              # Optional

    # Condition tree (optional — omit to match all data)
    condition:
      # Leaf condition
      leaf:
        type: "device_id"           # Condition type name
        config:
          values: ["sensor-01", "sensor-02"]

    # Actions (optional — chain evaluates condition only if omitted)
    actions:
      - type: "transform"
        config:
          scale: 1000
          offset: 0
          unit: "mV"
      - type: "limit_check"
        config:
          upper_limit: 250000
          lower_limit: 0

    # Routing targets (optional)
    targets: ["default"]

    # Input bindings (optional, for multi-input rules)
    input_bindings:
      voltage: "voltage"
      current: "current"
```

---

## Condition Trees

Conditions support AND/OR/NOT composition for complex logic:

```yaml
# AND: all children must match
condition:
  operator: AND
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

# OR: any child matching is sufficient
condition:
  operator: OR
  children:
    - leaf:
        type: "device_id"
        config:
          values: ["sensor-01"]
    - leaf:
        type: "device_id"
        config:
          values: ["sensor-02"]

# NOT: negate a condition
condition:
  operator: NOT
  children:
    - leaf:
        type: "quality"
        config:
          min_quality: 192

# Nested: AND inside OR
condition:
  operator: OR
  children:
    - operator: AND
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

## Condition Types Reference

| Type | Config Fields | Description |
|------|--------------|-------------|
| `device_type` | `values: [string]` | Match device type |
| `device_id` | `values: [string]` | Match device ID |
| `point_name` | `values: [string]` | Match point name |
| `point_name_pattern` | `pattern: string` | Regex match point name |
| `fqn_prefix` | `prefixes: [string]` | Match FQN prefix (Trie) |
| `value_range` | `min: float`, `max: float` | Numeric range filter |
| `value_in` | `values: [float]` | Discrete value match |
| `quality` | `min_quality: int` | Quality code filter |
| `limit_exceeded` | *(none)* | Current limit state |
| `time_window` | `start: string`, `end: string`, `timezone: string` | Time window |
| `state_change` | *(none)* | Detect value change |
| `dynamic_threshold` | `tag_key: string`, `operator: string` | Tag-based threshold |
| `expr_filter` | `expression: string` | expr-lang expression |
| `historical_compare` | *(see ext docs)* | Historical comparison |

---

## Action Types Reference

| Type | Config Fields | Description |
|------|--------------|-------------|
| `transform` | `scale: float`, `offset: float`, `unit: string` | Scale + offset |
| `rename` | *(none, reads `_rename` tag)* | Rename data point |
| `tag` | `tags: {string: string}` | Add key-value tags |
| `drop` | *(none)* | Discard data point |
| `route` | `targets: [string]` | Add routing targets |
| `limit_check` | `upper_limit: float`, `lower_limit: float` | Limit violation |
| `delay` | `delay: duration`, `action: {...}` | Async delayed action |
| `calc_node` | `expression: string` | expr-lang formula |
| `storage_write` | *(requires injection)* | Write to storage |
| `aggregation_write` | *(requires injection)* | Write aggregations |
| `device_aggregator` | *(requires injection)* | Device-level aggregation |
| `condition_expr_filter` | `expression: string` | Expr-based condition |

---

## Loading Configurations

### From file

```go
import (
    "github.com/wjffsx/ruleflow/pkg/ruleflow/config"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
    "github.com/wjffsx/ruleflow/pkg/ruleflow/builtin"
)

reg := nodes.NewEmptyRegistry()
builtin.RegisterAll(reg)

loader := config.DefaultLoader(reg, func(changed *core.RuleChain) {
    // called on each load/reload
    engine.LoadChain(changed)
})

chain, err := loader("path/to/chain.yaml")
```

### With file watcher (hot-reload)

```go
watcher := config.NewFileWatcher("path/to/chain.yaml", loader)
watcher.Start()
defer watcher.Stop()
```

### With directory watcher

```go
watcher := config.NewDirWatcher("path/to/chains/", loader)
watcher.Start()
defer watcher.Stop()
// Watches all *.yaml, *.yml, *.json files in the directory
```

---

## Validation

The configuration system performs these validations:

- Chain ID format (non-empty, printable ASCII)
- Status enum (`deployed`, `draft`, `disabled`)
- PipelineType enum (`analog`, `digital`, `meter`)
- Input deduplication (no duplicate point names)
- Input type consistency (same point_name must have same point_type)
- Rule ID uniqueness within a chain
- Input binding references must exist in `inputs`
- Condition tree depth limit (prevents stack overflow)
