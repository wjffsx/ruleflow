# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-15

### Added

- **Core Engine**
  - Compile-execute separation with function closure pre-compilation
  - Copy-on-write (COW) hot-reload with atomic snapshot swap
  - FastPath classification for sub-microsecond rule evaluation
  - Four-level backpressure control (Normal → Degraded → Paused → Dropping)
  - Pluggable error handling strategies (Continue, Abort, Retry, Fallback)
  - Object pooling for EvalResult (sync.Pool)

- **Contract Layer**
  - Zero-dependency interfaces: MetricsSink, Logger, Limiter, Tracer, Health
  - Compile-time interface checks for all noop implementations
  - Sentinel errors with errors.Is/errors.As support

- **Registry Architecture (V8 Refactoring)**
  - Interface layering: core.Registry, core.RegistryBuilder, core.RegistryQuerier
  - nodes.Registry implements core.FullRegistry
  - Type aliases for backward compatibility

- **Builtin Nodes**
  - Conditions: device_type, point_name, point_name_pattern, value_range, quality, limit_exceeded, device_id, fqn_prefix, value_in, time_window, rate_limit, state_change, duration, trend, periodic, dynamic_threshold
  - Actions: transform, rename, tag, drop, route, limit_check, quality_mark, delay, alarm_notify, bit_set, bit_check

- **Extension Nodes (ext package)**
  - Conditions: expr_filter, historical_compare
  - Actions: storage_write, aggregation_write, calc_node, alarm_notify_ext, quality_mark_ext, device_aggregator, strategy_execute, emit_soe, limit_tracker, meter_freeze, demand_calc

- **Domain-Specific Nodes (extensions package)**
  - VPP conditions: battery_soc, grid_frequency, power_factor, ramp_rate, market_price_threshold
  - VPP actions: aggregator, dispatch_control, market_price, carbon_calc, efficiency_calc, weather_integration
  - Flow nodes: msg_generator, sub_chain

- **Configuration**
  - YAML/JSON declarative rule chain definition
  - Two-phase parsing with validation and conflict detection
  - File watcher with debounce for hot-reload
  - Dependency graph for chain ordering

- **DataContext**
  - MapDataContext with thread-safe RWMutex
  - MultiDataContext for multi-input aggregation
  - MultiInputBuffer with pooling and timeout cleanup
  - DataContextAdapter for external system integration

- **Contrib Package**
  - Prometheus metrics sink
  - MemorySink for testing
  - slog adapter
  - OpenTelemetry tracer
  - TokenBucket rate limiter
  - CircuitBreaker with CAS state transition
  - pprof integration
  - Debug EventBus

### Fixed

- **Critical Issues**
  - MultiInputBuffer data race: async callback now releases pool object after completion
  - RateLimitCondition functionality: engine now saves timestamp to Tag
  - Object pool enablement: newEvalResult supports pooled allocation
  - MapDataContext thread safety: all fields protected by RWMutex
  - RenameAction: DataContext interface now includes SetPointName

- **Warning-Level Issues**
  - FastPath type mapping privatized with read-only access functions
  - Conflict detection algorithm upgraded with value_range overlap check
  - value_in condition uses sorted array + binary search (avoids float64 map key issues)
  - DataContextAdapter.SetPreviousValue uses value field (no heap allocation)
  - Timestamp unit judgment uses 1e15 threshold
  - ErrResource sentinel error added
  - removeFromSlice creates new slice (no underlying array modification)
  - FileWatcher debounce unified with time.AfterFunc
  - ext package factory functions use comma-ok type assertion

- **Info-Level Improvements**
  - MaxMetadataSize extracted as constant
  - MemorySink uses RWMutex instead of Mutex
  - CircuitBreaker uses CAS for state transition atomicity
  - Compile-time interface checks for all noop implementations

### Performance

- Hot path (Evaluate/Execute) achieves **0 B/op, 0 allocs/op**
- FastPath rules evaluate in <200ns
- COW snapshot swap is lock-free on read path

### Documentation

- Architecture diagram in README
- Custom nodes guide in docs/custom-nodes.md
- Configuration reference in docs/configuration.md
- Code review and optimization plan in docs/code-review-and-optimization.md
- bilingual README (English + Chinese)

### Testing

- Unit tests with race detector
- Integration tests gated with `//go:build integration`
- Fuzz tests for core engine
- Benchmarks for hot path verification
- CI workflow with coverage reporting