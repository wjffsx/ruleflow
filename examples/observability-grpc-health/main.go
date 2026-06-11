// Package main 演示如何直接基于 core/health 接口实现 gRPC HealthService。
//
// 这是 contrib/health/grpc 的推荐替代写法（V3.8 拆分自 observability-grpc）：
//   - 库只提供"能力（接口）"——不绑定具体协议
//   - 应用层负责"协议暴露"——30-50 行代码即可
//
// DebugService 演示见 examples/observability-grpc-debug。
//
// 端点：
//   - gRPC HealthService  - 标准 grpc.health.v1 健康检查（liveness/readiness）
//
// 运行：
//
//	go run examples/observability-grpc-health/main.go
//
// 客户端测试：
//
//	使用 grpcurl 或自定义 gRPC 客户端
package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/health/grpc_health_v1"
)

// ⚠️ 本示例为教学演示。生产代码应使用：
//   - TLS / 鉴权
//   - 优雅关闭
//   - 拦截器（metrics / logging）

func main() {
	eng := engine.NewEngine()
	defer eng.Shutdown(context.Background())

	// 启动 gRPC server
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// 注册 HealthService（基于 core/health 接口实现）
	pb.RegisterHealthServer(grpcServer, &grpcHealthServer{engine: eng})

	log.Printf("observability-grpc-health listening on :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// ─────────────────────────────────────────────
//  HealthService gRPC 实现
//  基于 core/health 接口，最小可工作实现
// ─────────────────────────────────────────────

// grpcHealthServer 实现 grpc.health.v1.HealthServer
type grpcHealthServer struct {
	pb.UnimplementedHealthServer
	engine *engine.Engine
}

// Check 实现标准 gRPC health check（用于 Kubernetes liveness/readiness）
func (s *grpcHealthServer) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// 根据 service 名字判断 liveness / readiness
	switch req.Service {
	case "ruleflow.readiness":
		ready, _ := s.engine.IsReady(ctx)
		if ready {
			return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
		}
		return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_NOT_SERVING}, nil
	default:
		// liveness
		if s.engine.IsAlive(ctx) {
			return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
		}
		return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_NOT_SERVING}, nil
	}
}

// Watch 实现流式健康检查（gRPC 标准）
func (s *grpcHealthServer) Watch(req *pb.HealthCheckRequest, stream pb.Health_WatchServer) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			resp, err := s.Check(stream.Context(), req)
			if err != nil {
				return err
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}
