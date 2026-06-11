# observability-grpc-health

演示如何直接基于 `core/health` 接口实现 gRPC `HealthService`。

## 这是什么

V3.8 起，从 `examples/observability-grpc` 拆出。HealthService 是最常见的 gRPC 服务，单独示例化便于复制。

- 库只提供 **能力（接口）** —— 不绑定具体协议
- 应用层负责 **协议暴露** —— 30-50 行代码即可
- TLS、鉴权、拦截器 —— 全部留给用户决定

## DebugService 怎么办？

DebugService（gRPC Server Streaming）见姊妹示例：

- [`examples/observability-grpc-debug/`](../observability-grpc-debug/README.md)

## 端点

| Service  | RPC     | 用途                                    | 底层接口                       |
| -------- | ------- | --------------------------------------- | ------------------------------ |
| `Health` | `Check` | 标准 gRPC health（liveness/readiness）  | `core/health.LivenessChecker` 等 |
| `Health` | `Watch` | 流式健康检查（5s 间隔）                 | `core/health.LivenessChecker` 等 |

## 运行

```bash
cd ruleflow
go run examples/observability-grpc-health/main.go
```

服务默认监听 `:9090`。

## 测试

### liveness（默认 service 名为空字符串）

```bash
grpcurl -plaintext -import-path . -proto google.golang.org/grpc/health/grpc_health_v1/health.proto \
    -d '{"service": ""}' \
    :9090 grpc.health.v1.Health/Check
```

### readiness（service 名 = `ruleflow.readiness`）

```bash
grpcurl -plaintext -import-path . -proto google.golang.org/grpc/health/grpc_health_v1/health.proto \
    -d '{"service": "ruleflow.readiness"}' \
    :9090 grpc.health.v1.Health/Check
```

### 流式 liveness

```bash
grpcurl -plaintext -import-path . -proto google.golang.org/grpc/health/grpc_health_v1/health.proto \
    -d '{"service": ""}' \
    :9090 grpc.health.v1.Health/Watch
```

## 关键代码

### HealthService（约 30 行）

```go
type grpcHealthServer struct {
    pb.UnimplementedHealthServer
    engine *engine.Engine
}

func (s *grpcHealthServer) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
    switch req.Service {
    case "ruleflow.readiness":
        ready, _ := s.engine.IsReady(ctx)
        if ready {
            return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
        }
        return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_NOT_SERVING}, nil
    default:
        if s.engine.IsAlive(ctx) {
            return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
        }
        return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_NOT_SERVING}, nil
    }
}
```

## 进阶：生产环境建议

示例代码故意保持简洁。生产环境通常需要：

- **TLS**：`grpc.Creds(creds.NewServerTLSFromCert(...))`
- **鉴权**：Token-based / mTLS
- **拦截器**：metrics / logging / tracing
- **优雅关闭**：`grpc.GracefulStop()` + context 超时

这些都应在应用层完成，**不应**侵入 ruleflow 库。

## 对应废弃包

| 旧 import                                                    | 状态         | 替代                       |
| ------------------------------------------------------------ | ------------ | -------------------------- |
| `github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/health/grpc` | ⚠️ Deprecated | 本示例的 `HealthService`   |
