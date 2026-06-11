// Package contract defines the contract layer for the ruleflow engine.
//
// This package contains pure interfaces, enumerations, value types, and
// zero-logic noop default implementations. The noop implementations
// (NoopLogger, NoopSink, NoopLimiter, etc.) and function adapters
// (FuncProvider, FuncTracerProvider) are included as "zero-value usable"
// defaults — they contain no business state and serve only as safe
// fallbacks when no concrete implementation is provided.
//
// Dependency direction:
//   - core/contract  ← core/engine
//   - core/contract  ← core/compiler
//   - core/contract  ← contrib/* (adapter layer)
//   - core/contract  ← nodes/* (node layer)
//
// This package must NOT import any other ruleflow internal package.
package contract
