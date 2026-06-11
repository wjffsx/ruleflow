# Contributing to RuleFlow

Thank you for considering contributing to RuleFlow. This document outlines the development workflow, code conventions, and pull request process.

---

## Development Setup

### Requirements

- Go 1.24+
- Git
- (Optional) golangci-lint v1.59+ for local linting

### Clone and build

```bash
git clone https://github.com/wjffsx/ruleflow.git
cd ruleflow
go mod download
go build ./...
```

### Run tests

```bash
# Unit tests with race detector
go test -count=1 -race ./pkg/... ./benchmark/...

# Integration tests (requires -tags=integration)
go test -tags=integration -count=1 -timeout 300s ./tests/...

# Fuzz tests (60s per target)
go test -fuzz=FuzzLimiterKey -fuzztime=60s ./pkg/ruleflow/core/
go test -fuzz=FuzzEvalWithRandomData -fuzztime=60s ./pkg/ruleflow/core/
go test -fuzz=FuzzPanicRecovery -fuzztime=60s ./pkg/ruleflow/core/

# Run benchmarks
go test -bench=. -benchtime=1s -run=^$ ./benchmark/...
```

### Run linter

```bash
golangci-lint run --timeout=10m ./...
```

---

## Code Conventions

### General

- Use `go fmt` before committing (the CI enforces this via golangci-lint).
- Use meaningful variable names. Single-letter names are acceptable only in very short scopes (e.g., loop counters).
- Prefer early returns over deep nesting.
- Keep functions short: if a function exceeds 40 lines, consider extracting helper functions.

### Performance

- The hot path (`Evaluate` / `Execute` / `EvalChain`) must remain **zero heap allocation**. Use `-benchmem` and `go test -bench` to verify.
- Use `sync.Pool` for frequently allocated transient objects.
- Pre-compute in factory functions; avoid computation in `Evaluate`/`Execute`.
- The `DataContext` interface intentionally avoids returning slices or maps — follow this pattern for new interfaces on the hot path.

### Error handling

- Use `errors.New()` for static messages, `fmt.Errorf()` for dynamic messages.
- Use `errors.Is()` / `errors.As()` for error comparison — never `==`.
- Sentinel errors go in `core/errors.go`.
- New error types should embed `RuleFlowError` or at least implement `Unwrap()`.

### Testing

- Tests go in `*_test.go` files alongside the code they test.
- Use `t.Parallel()` for independent test cases.
- Table-driven tests are preferred over imperative test functions.
- Integration tests go in `tests/integration/` and are gated with `//go:build integration`.
- Fuzz targets start with `Fuzz` and live in `*_test.go` files.

### Documentation

- Export all public types and functions with Go doc comments.
- Document config fields in `meta.go` files using `ComponentMeta`.
- Examples go in `examples/` — each example must have a `README.md` explaining its purpose.

---

## Pull Request Process

1. **Fork the repository** and create a feature branch from `main`.
2. **Write tests** for new functionality. Bug fixes should include a regression test.
3. **Run the full test suite** locally before pushing.
4. **Keep PRs focused** — one feature or fix per PR. Large changes should be discussed in an issue first.
5. **Write a clear PR description** explaining what changed and why.
6. **CI will run automatically** on push. All checks must pass before merge.

### PR title format

```
{type}: {short description}
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

Examples:
- `feat: add sliding window rate limiter condition`
- `fix: prevent race in MultiInputBuffer cleanup`
- `docs: add custom nodes guide`

---

## Branch Strategy

- `main` — stable, released. All commits must pass CI.
- Feature branches — `feat/{description}`, branched from `main`.
- Bug fixes — `fix/{description}`, branched from `main`.

There is no `develop` branch. Feature branches merge directly to `main` via PR.

---

## Adding a New Node

1. Choose the right package:
   - `builtin/` — IoT-generic, zero external dependencies
   - `ext/` — IoT-extension, requires dependency injection
   - `extensions/` — VPP/energy domain-specific

2. Implement `Condition` or `Action` interface.
3. Add a factory function.
4. Register in the package's `register.go`.
5. Add tests (unit + benchmark if on hot path).
6. Add `ComponentMeta` for tooling.
7. Document in `docs/custom-nodes.md` or `docs/configuration.md`.

---

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.
