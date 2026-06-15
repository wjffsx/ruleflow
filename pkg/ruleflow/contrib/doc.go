// Package contrib 提供可选的外部依赖实现。
//
// V12 架构定位说明：
//   - 本包是可选的扩展层模块，不属于核心库
//   - 提供外部依赖的具体实现（OpenTelemetry、Prometheus、日志等）
//   - 应用层按需选择启用，不强制依赖
//
// 子包功能分组（建议）：
//
// ┌─────────────────────────────────────────────────────────────┐
// │  可观测性 (observability)                                   │
// │  - otel:      OpenTelemetry 集成                            │
// │  - prometheus: Prometheus 指标导出                          │
// │  - slog:      结构化日志集成                                 │
// │  - pprof:     Go pprof 性能分析端点                          │
// │  - profiler:  自定义性能分析器                               │
// ├─────────────────────────────────────────────────────────────┤
// │  容错机制 (resilience)                                      │
// │  - circuitbreaker: 熔断器实现                               │
// │  - tokenbucket:    令牌桶限流器                              │
// ├─────────────────────────────────────────────────────────────┤
// │  存储实现 (storage)                                         │
// │  - memorystate: 内存状态存储（用于有状态条件）               │
// ├─────────────────────────────────────────────────────────────┤
// │  调试工具 (debug)                                           │
// │  - memorysink:   内存调试输出                               │
// │  - ratelimit_sink: 限流调试输出                             │
// │  - eventbus:     调试事件总线                               │
// └─────────────────────────────────────────────────────────────┘
//
// 依赖方向：
//   - contrib/* → core/contract（正确）
//   - contrib/* 不依赖 builtin/ext/extensions（正确）
//
// 使用建议：
//   - 生产环境：按需选择 otel/prometheus/slog
//   - 开发环境：使用 memorysink/memorystate 进行调试
//   - 高可用场景：使用 circuitbreaker/tokenbucket 增强容错
package contrib
