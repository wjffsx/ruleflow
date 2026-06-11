# Custom Components Example

Demonstrates how to implement and register custom `Condition` and `Action` nodes with application-specific domain logic.

## Run

```bash
go run main.go
```

## What it covers

- **Custom Condition**: `MaintenanceWindowCondition` — restricts rule evaluation to a configurable time window (e.g., 9:00–18:00).
- **Custom Action**: `WebhookAction` — pushes limit-exceeded events to an external HTTP endpoint.
- **MapDataContext**: Using `datacontext.NewMapDataContext()` as a lightweight DataContext for simple use cases — no need to implement the full interface.
- **Custom node registration**: Register custom types with `nodes.Registry` and pass the registry to engine options.

## Key takeaway

To add custom logic, implement `core.Condition` / `core.Action` interfaces, register via `nodes.NewEmptyRegistry()`, and inject into the engine via `engine.WithRegistry()`.
