# Basic Example

Minimal RuleFlow setup demonstrating the core workflow:

- Creating an engine
- Building a rule chain programmatically (code-based, no YAML)
- Implementing a custom `DataContext`
- Evaluating data points through the engine
- Understanding matched vs dropped results

## Run

```bash
go run main.go
```

## What it does

1. Creates two rules in a chain:
   - **filter_analog**: Matches analog data points, applies a 10× transform (scale=0.1), checks limits, and marks quality.
   - **drop_digital**: Drops digital data points entirely.
2. Evaluates an analog data point (`voltage=220.5`) — matches rule 1, value transforms to 22.05.
3. Evaluates a digital data point (`switch=1.0`) — matches rule 2, gets dropped.

## Key takeaway

The simplest way to use RuleFlow is: `NewEngine()` → `LoadChain(chain)` → `EvalChain(ctx, id, data)`.
