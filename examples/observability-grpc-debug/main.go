// Package main 演示如何基于 contrib/debug.ChannelSink 实现 gRPC DebugService 骨架。
//
// 这是 contrib/debug/grpc 的推荐替代写法（V3.8 拆分自 observability-grpc）：
//   - 库只提供"能力（ChannelSink）"——不绑定具体协议
//   - 应用层负责"协议暴露"——本示例展示核心模式（30-50 行）
//
// HealthService 演示见 examples/observability-grpc-health。
//
// ⚠️ 本示例为骨架演示：
//   - 定义了 .proto 接口的 Go struct 等价物（debugServer）+ RPC 方法
//   - 未生成 .pb.go（protoc 生成的代码不入仓）
//   - Subscribe/SetDebugMode 给出"接 ChannelSink"的真实可运行代码
//   - 完整 .proto 模板见 README
//
// 运行：
//
//	go run examples/observability-grpc-debug/main.go
package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/contrib/debug"
	coredebug "github.com/vpptu/ruleflow/pkg/ruleflow/debug"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/engine"

	"google.golang.org/grpc"
)

func main() {
	eng := engine.NewEngine()
	defer eng.Shutdown(context.Background())

	// V4.10：创建 ChannelSink（替代 EventBus）—— *ChannelSink 自身实现 coredebug.DebugSink
	sink := debug.NewChannelSink(1024)

	// 注入到引擎（可选 - 仅在需要调试时）：
	//   engine.WithDebugManager(coredebug.NewDebugManager(coredebug.DebugAll, sink, time.Time{}))
	// 直接传 sink，无需 NewEventBusSink 包装（V4.3 删除冗余包装）

	// 启动 gRPC server
	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// 完整代码中应通过 protoc 生成的 pb.RegisterDebugServiceServer 注册：
	//   pb.RegisterDebugServiceServer(grpcServer, &debugServer{sink: sink, engine: eng})
	// 这里直接演示核心模式（跳过 pb 注册）
	_ = &debugServer{sink: sink, engine: eng}

	log.Printf("observability-grpc-debug listening on :9091 (DebugService skeleton; pb registration requires protoc-generated code)")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// ─────────────────────────────────────────────
//  DebugService gRPC 实现（骨架）
//  基于 contrib/debug.ChannelSink，Server-Side Streaming
// ─────────────────────────────────────────────

// debugServer 模拟 DebugService 的 gRPC 实现。
//
// 完整实现需要：
//  1. 定义 .proto 文件（DebugService with Subscribe, SetDebugMode RPCs）
//  2. 生成 gRPC 代码（protoc-gen-go + protoc-gen-go-grpc）
//  3. 替换本骨架为 pb.RegisterDebugServiceServer(...)
type debugServer struct {
	// pb.UnimplementedDebugServiceServer
	sink   *debug.ChannelSink
	engine *engine.Engine
}

// SubscribeRequest / SubscribeResponse / SetDebugModeRequest / SetDebugModeResponse
// 是 .proto 模板的 Go struct 等价物（仅作类型占位）。
type SubscribeRequest struct {
	ChainIDs   []string
	RuleIDs    []string
	NodeType   string
	OnlyErrors bool
}

type SubscribeResponse struct {
	Event coredebug.DebugEvent
}

type SetDebugModeRequest struct {
	Mode string // "off" | "failures" | "all"
}

type SetDebugModeResponse struct {
	Success bool
}

// Subscribe 实现 gRPC Server-Side Streaming，从 ChannelSink 拉取事件。
//
// 核心模式：
//   - 每个客户端独立 channel（防背压）
//   - 启动 goroutine 从 sink.Subscribe() 拉取事件
//   - 推送到客户端 gRPC stream
func (s *debugServer) Subscribe(_ context.Context, req *SubscribeRequest, send func(SubscribeResponse) error) error {
	// 实际使用 req 构建 SubscribeFilter（见 internal/filter.go）
	_ = req

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 每个客户端独立 channel（防背压）
	ch := make(chan coredebug.DebugEvent, 256)
	defer close(ch)

	// 启动 goroutine 从 ChannelSink 读取
	go func() {
		for event := range s.sink.Subscribe() {
			select {
			case ch <- event:
			default:
				// buffer 满，丢弃（背压防护）
			}
		}
	}()

	// 流式推送到客户端
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if err := send(SubscribeResponse{Event: event}); err != nil {
				return err
			}
		}
	}
}

// SetDebugMode 实现切换调试模式（Off / Failures / All）
func (s *debugServer) SetDebugMode(_ context.Context, req *SetDebugModeRequest) (SetDebugModeResponse, error) {
	dm := s.engine.GetDebugManager()
	if dm == nil {
		return SetDebugModeResponse{Success: false}, nil
	}
	// allDeadline 留零值，模式立即生效
	switch req.Mode {
	case "off":
		dm.SetMode(coredebug.DebugOff, time.Time{})
	case "failures":
		dm.SetMode(coredebug.DebugFailures, time.Time{})
	case "all":
		dm.SetMode(coredebug.DebugAll, time.Time{})
	default:
		return SetDebugModeResponse{Success: false}, nil
	}
	return SetDebugModeResponse{Success: true}, nil
}
