package engine

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  HealthCheck — 健康检查实现
// ─────────────────────────────────────────────

// V13 架构边界说明：
//   - 本文件属于核心库，提供 Engine 的健康检查和关闭功能
//   - 只依赖 contract 包，符合核心库零依赖要求
//   - HealthCheck/Shutdown 是 Engine 的核心方法，不应移出
//
// 与 contrib 的区别：
//   - core/engine: 核心健康检查和关闭逻辑
//   - contrib/health: 可选的健康检查端点（HTTP/gRPC）

// HealthCheck 返回引擎健康状态（直接返回 contract.HealthStatus）
//
// V4.4 收敛：
//   - 删除 MemoryUsage / SysMemoryUsage / NumGoroutine 三个字段填充（runtime 越界）
//   - 删除 LastError 字段填充（业务错误状态越界，由应用层自行记录）
//   - UptimeSeconds / Uptime 由 startTimeFn 注入
func (e *Engine) HealthCheck() contract.HealthStatus {
	shuttingDown := e.shutdown.isShuttingDown()

	status := "healthy"
	if shuttingDown {
		status = "unhealthy"
	}

	uptime := e.uptimeFn()
	hs := contract.HealthStatus{
		Status:        status,
		LoadedChains:  len(e.GetChainIDs()),
		ActiveEval:    atomic.LoadInt64(&e.activeEval),
		ShuttingDown:  shuttingDown,
		UptimeSeconds: int64(uptime.Seconds()),
		Uptime:        uptime.String(),
		Timestamp:     time.Now(),
	}

	return hs
}

// IsAlive 实现 contract.LivenessChecker 接口
func (e *Engine) IsAlive(_ context.Context) bool {
	return !e.shutdown.isShuttingDown()
}

// IsReady 实现 contract.ReadinessChecker 接口
func (e *Engine) IsReady(_ context.Context) (bool, contract.HealthStatus) {
	hs := e.HealthCheck()
	if hs.ShuttingDown || hs.LoadedChains == 0 {
		return false, hs
	}
	return true, hs
}

// ReportStatus 实现 contract.StatusReporter 接口
func (e *Engine) ReportStatus(_ context.Context) contract.HealthStatus {
	return e.HealthCheck()
}

// ListChains 实现 contract.ChainLister 接口
func (e *Engine) ListChains(_ context.Context) ([]string, error) {
	return e.GetChainIDs(), nil
}

// ─────────────────────────────────────────────
//  Shutdown — 优雅关闭
// ─────────────────────────────────────────────

// Shutdown 优雅关闭引擎。
func (e *Engine) Shutdown(ctx context.Context) error {
	if !e.shutdown.markShutdown() {
		return nil
	}
	e.logger.Info("engine shutdown initiated")
	err := e.shutdown.wait(ctx)
	if err != nil {
		e.logger.Error("engine shutdown timeout", "err", err)
	} else {
		e.logger.Info("engine shutdown complete")
	}
	return err
}

// StartTime 返回引擎构造时刻
//
// V4.7：startTime 字段改为方法（构造时通过闭包注入 startTimeFn）。
func (e *Engine) StartTime() time.Time {
	return e.startTimeFn()
}
