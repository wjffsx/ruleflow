# observability-grpc-debug

演示如何基于 `contrib/debug.EventBus` 实现 gRPC `DebugService` 骨架。

## 这是什么

V3.8 起，从 `examples/observability-grpc` 拆出。DebugService 涉及 gRPC Server-Side Streaming 模式，且需先有 `.proto` 文件生成的代码。单独示例化便于专注于"接 EventBus"的模式。

- 库只提供 **能力（EventBus）** —— 不绑定具体协议
- 应用层负责 **协议暴露** —— 30-50 行代码即可
- TLS、鉴权、拦截器 —— 全部留给用户决定

## HealthService 怎么办？

HealthService 演示见姊妹示例：

- [`examples/observability-grpc-health/`](../observability-grpc-health/README.md)

## 端点

| Service          | RPC            | 用途                                    | 底层接口                     |
| ---------------- | -------------- | --------------------------------------- | ---------------------------- |
| `Debug` (待生成) | `Subscribe`    | 调试事件 Server Streaming              | `contrib/debug.EventBus`    |
| `Debug` (待生成) | `SetDebugMode` | 切换调试模式（Off/Failures/All）        | `core/debug.DebugManager`   |

## 运行

```bash
cd ruleflow
go run examples/observability-grpc-debug/main.go
```

服务默认监听 `:9091`（避免与 `observability-grpc-health` 冲突）。

## 生成 .proto 代码

DebugService 需要先定义 `.proto` 并生成代码。完整 `.proto` 模板：

```protobuf
syntax = "proto3";
package debug.v1;

import "google/protobuf/timestamp.proto";

service DebugService {
    // 订阅调试事件流（Server Streaming）
    rpc Subscribe(SubscribeRequest) returns (stream SubscribeResponse);
    // 切换调试模式
    rpc SetDebugMode(SetDebugModeRequest) returns (SetDebugModeResponse);
}

message SubscribeRequest {
    repeated string chain_ids = 1;
    repeated string rule_ids  = 2;
    string node_type          = 3;
    bool   only_errors        = 4;
}

message SubscribeResponse {
    DebugEvent event = 1;
}

message SetDebugModeRequest {
    string mode = 1; // "off" | "failures" | "all"
}

message SetDebugModeResponse {
    bool success = 1;
}

message DebugEvent {
    string chain_id       = 1;
    string rule_id        = 2;
    string node_id        = 3;
    string node_type      = 4;
    string event_type     = 5;
    string relation_type  = 6;
    string data_snapshot  = 7;
    google.protobuf.Timestamp timestamp = 8;
    int64  duration_ns    = 9;
}
```

生成 Go 代码：

```bash
protoc --go_out=. --go-grpc_out=. debug.proto
```

然后替换 `main.go` 中"骨架"标记：

```go
// 替换：
pb.RegisterDebugServiceServer(grpcServer, &debugServer{bus: bus, engine: eng})
```

## 关键模式

### Subscribe（Server Streaming）

```go
func (s *debugServer) Subscribe(_ context.Context, req *SubscribeRequest, stream pb.DebugService_SubscribeServer) error {
    ctx, cancel := context.WithCancel(stream.Context())
    defer cancel()

    // 1. 每个客户端独立 channel（防背压）
    ch := make(chan coredebug.DebugEvent, 256)
    defer close(ch)

    // 2. 用 filter 过滤（见 internal/filter.go）
    filter := debuginternal.NewSubscribeFilter(req.ChainIds, req.RuleIds, req.NodeType, req.OnlyErrors)

    // 3. 启动 goroutine 从 EventBus 读取
    go func() {
        for event := range s.bus.Subscribe() {
            if !filter.Match(event) {
                continue
            }
            select {
            case ch <- event:
            default:
                // buffer 满，丢弃（背压防护）
            }
        }
    }()

    // 4. 流式推送到客户端
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case event, ok := <-ch:
            if !ok {
                return nil
            }
            if err := stream.Send(&pb.SubscribeResponse{
                Event: convertToProto(event),
            }); err != nil {
                return err
            }
        }
    }
}
```

### SetDebugMode

```go
func (s *debugServer) SetDebugMode(_ context.Context, req *pb.SetDebugModeRequest) (*pb.SetDebugModeResponse, error) {
    dm := s.engine.GetDebugManager()
    if dm == nil {
        return &pb.SetDebugModeResponse{Success: false}, nil
    }
    switch req.Mode {
    case "off":
        dm.SetMode(coredebug.DebugOff)
    case "failures":
        dm.SetMode(coredebug.DebugFailures)
    case "all":
        dm.SetMode(coredebug.DebugAll)
    default:
        return &pb.SetDebugModeResponse{Success: false}, nil
    }
    return &pb.SetDebugModeResponse{Success: true}, nil
}
```

## 进阶：生产环境建议

- **TLS**：`grpc.Creds(creds.NewServerTLSFromCert(...))`
- **鉴权**：Token-based / mTLS
- **拦截器**：metrics / logging / tracing
- **优雅关闭**：`grpc.GracefulStop()` + context 超时
- **完整 .proto**：支持 filter、batch、ack

这些都应在应用层完成，**不应**侵入 ruleflow 库。

## 对应废弃包

| 旧 import                                                | 状态         | 替代                       |
| -------------------------------------------------------- | ------------ | -------------------------- |
| `github.com/vpptu/ruleflow/pkg/ruleflow/contrib/debug/grpc` | ⚠️ Deprecated | 本示例的 `DebugService`   |
| `github.com/vpptu/ruleflow/pkg/ruleflow/contrib/debug/sse`  | ⚠️ Deprecated | 应用层 SSE/WebSocket    |
