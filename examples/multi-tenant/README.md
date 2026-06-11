# Multi-Tenant Example

Demonstrates per-tenant rule isolation with independent rate limiters for each tenant.

## Run

```bash
go run main.go
```

## What it covers

- **Per-tenant engine instances**: Each tenant gets its own `Engine` instance with independent rule chains.
- **Per-tenant rate limiting**: Uses `tokenbucket.PerKeyLimiter` wrapped in a `tenantLimiter` — each tenant has its own token bucket to prevent one noisy tenant from starving others.
- **Contract-based limiter**: The limiter implements `contract.Limiter` and is injected via `engine.WithLimiter()`.
- **Fair resource sharing**: Each tenant's data is processed independently, with metrics and limits scoped per tenant key.

## Key takeaway

For multi-tenant deployments, create one engine per tenant and inject per-tenant dependencies (limiters, metrics sinks, state stores) through engine options. This provides strong isolation without shared state.
