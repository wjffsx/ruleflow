// Package main 演示如何直接基于 contract.HealthProvider + contract.MetricsSink
// 实现 HTTP 可观测性端点。
//
// 这是 httphealth contrib 包的推荐替代写法：
//   - 库只提供"能力（接口 + sink）"——不绑定具体协议
//   - 应用层负责"协议暴露"——30-50 行代码即可
//
// 端点：
//   - GET /healthz   - 存活检查
//   - GET /readyz    - 就绪检查
//   - GET /status    - 详细健康状态
//   - GET /metrics   - 引擎指标快照（来自 contract.MetricsSink，含节点级聚合）
//   - GET /chains    - 已加载链列表
//
// 运行：
//
//	go run examples/observability-http/main.go
//
// 测试：
//
//	curl http://localhost:8080/healthz
//	curl http://localhost:8080/readyz
//	curl http://localhost:8080/status
//	curl http://localhost:8080/metrics
//	curl http://localhost:8080/chains
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/contrib/memorysink"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/engine"
)

func main() {
	// 注入内存 sink 用于 /metrics
	sink := memorysink.NewMemorySink()

	eng := engine.NewEngine(engine.WithMetricsSink(sink))
	defer eng.Shutdown(context.Background())

	mux := http.NewServeMux()

	// 引擎 *engine.Engine 已实现 contract 包的 4 个接口：
	//   - contract.LivenessChecker
	//   - contract.ReadinessChecker
	//   - contract.StatusReporter
	//   - contract.ChainLister
	mux.HandleFunc("/healthz", livenessHandler(eng))
	mux.HandleFunc("/readyz", readinessHandler(eng, 1))
	mux.HandleFunc("/status", statusHandler(eng))
	// V4.5 收敛：节点级性能聚合通过 /metrics 的 ConditionEval / ActionExec 提供；
	// 不再单独暴露 /profiler 端点（避免与 memorysink 重复）。
	mux.HandleFunc("/metrics", metricsHandler(sink))
	mux.HandleFunc("/chains", chainsHandler(eng))

	addr := ":8080"
	log.Printf("observability-http listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// livenessHandler 存活检查（5 行）
func livenessHandler(hc contract.LivenessChecker) http.HandlerFunc {
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

// readinessHandler 就绪检查（10 行）
func readinessHandler(rc contract.ReadinessChecker, minChains int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ready, hs := rc.IsReady(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if !ready || hs.LoadedChains < minChains {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":        "not_ready",
				"shutting_down": hs.ShuttingDown,
				"loaded_chains": hs.LoadedChains,
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":        "ready",
			"loaded_chains": hs.LoadedChains,
		})
	}
}

// statusHandler 详细健康状态（10 行）
func statusHandler(sr contract.StatusReporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hs := sr.ReportStatus(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hs)
	}
}

// metricsHandler 指标快照（直接来自 memorysink）
func metricsHandler(sink *memorysink.MemorySink) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := sink.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(snap)
	}
}

// chainsHandler 链列表
func chainsHandler(cl contract.ChainLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		ids, err := cl.ListChains(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chain_ids": ids,
			"count":     len(ids),
		})
	}
}
