# observability-http

演示如何直接基于 `core/health` 接口实现 HTTP 可观测性端点。

## 这是什么

这是 `contrib/observability/httphealth/` 包的 **推荐替代写法**：

- 库只提供 **能力（接口）** —— 不绑定具体协议
- 应用层负责 **协议暴露** —— 5-10 行代码即可
- 鉴权、限流、租户隔离、面板路径 —— 全部留给用户决定

## 端点

| 端点         | 方法 | 用途                  | 实现接口                          |
| ------------ | ---- | --------------------- | --------------------------------- |
| `/healthz`   | GET  | 存活检查（liveness）  | `health.LivenessChecker`          |
| `/readyz`    | GET  | 就绪检查（readiness） | `health.ReadinessChecker`         |
| `/status`    | GET  | 详细健康状态          | `health.StatusReporter`           |
| `/metrics`   | GET  | 引擎指标快照（JSON）  | `health.MetricsProvider`          |
| `/profiler`  | GET  | 节点性能排行          | `health.ProfilerProvider`         |
| `/chains`    | GET  | 已加载链 ID 列表      | `health.ChainLister`              |

## 运行

```bash
cd ruleflow
go run examples/observability-http/main.go
```

服务默认监听 `:8080`。

## 测试

```bash
# 存活检查
curl -i http://localhost:8080/healthz

# 就绪检查
curl -i http://localhost:8080/readyz

# 详细健康状态
curl http://localhost:8080/status | jq

# 指标快照
curl http://localhost:8080/metrics | jq

# 节点性能排行（按最大延迟）
curl 'http://localhost:8080/profiler?top=5&sort=max_latency' | jq

# 节点性能排行（按平均延迟）
curl 'http://localhost:8080/profiler?top=5&sort=avg_latency' | jq

# 节点性能排行（按执行次数）
curl 'http://localhost:8080/profiler?top=5&sort=exec_count' | jq

# 已加载链 ID
curl http://localhost:8080/chains | jq
```

## 关键代码

每个 handler 都是 5-10 行：

```go
// livenessHandler 存活检查（5 行核心逻辑）
func livenessHandler(hc health.LivenessChecker) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        alive := hc.IsAlive(r.Context())
        w.Header().Set("Content-Type", "application/json")
        if alive {
            w.WriteHeader(http.StatusOK)
            _ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
            _ = json.NewEncoder(w).Encode(map[string]string{"status": "dead"})
        }
    }
}
```

引擎 `*engine.Engine` 已经实现了 `core/health` 包的 **所有** 接口，因此可以直接传入。

## 进阶：生产环境建议

示例代码故意保持简洁。生产环境通常需要：

- **TLS**：使用 `http.ListenAndServeTLS` 或反向代理
- **鉴权**：JWT / Basic Auth / mTLS
- **限流**：`golang.org/x/time/rate` 或中间件
- **多租户隔离**：按租户 ID 过滤 `Snapshot()` 输出
- **面板格式**：把 `/metrics` 改为 Prometheus 格式（用 `contrib/prometheus`）
- **链路追踪**：用 `otelhttp` 中间件包装

这些都应在应用层完成，**不应**侵入 ruleflow 库。

## 对应废弃包

| 旧 import                                                              | 状态         | 替代                                       |
| ---------------------------------------------------------------------- | ------------ | ------------------------------------------ |
| `github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/observability/httphealth` | ⚠️ Deprecated | 本示例（5-10 行/handler）                  |
