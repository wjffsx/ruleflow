package circuitbreaker

import (
	"sync/atomic"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  CircuitBreaker — 熔断器适配器
// ─────────────────────────────────────────────

// CircuitState 熔断器状态
type CircuitState int32

const (
	CircuitClosed   CircuitState = iota // 正常
	CircuitOpen                         // 熔断
	CircuitHalfOpen                     // 半开
)

// CircuitBreaker 熔断器，实现 BackpressureIndicator 接口
type CircuitBreaker struct {
	failureThreshold int           // 连续失败阈值
	recoveryTimeout  time.Duration // 熔断恢复时间
	halfOpenRequests int           // 半开状态测试请求数

	state        atomic.Int32 // CircuitState
	failureCount atomic.Int64
	lastFailure  atomic.Int64 // unix nano
	successCount atomic.Int64
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
		halfOpenRequests: 3,
	}
}

// ShouldAccept 实现 BackpressureIndicator 接口
func (cb *CircuitBreaker) ShouldAccept(deviceID string) bool {
	switch CircuitState(cb.state.Load()) {
	case CircuitClosed:
		return true
	case CircuitOpen:
		lastFail := time.Unix(0, cb.lastFailure.Load())
		if time.Since(lastFail) > cb.recoveryTimeout {
			cb.state.Store(int32(CircuitHalfOpen))
			cb.successCount.Store(0)
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.successCount.Load() < int64(cb.halfOpenRequests)
	default:
		return true
	}
}

// CurrentLevel 实现 BackpressureIndicator 接口
func (cb *CircuitBreaker) CurrentLevel() contract.Level {
	switch CircuitState(cb.state.Load()) {
	case CircuitClosed:
		return contract.Normal
	case CircuitOpen:
		return contract.Dropping
	case CircuitHalfOpen:
		return contract.Degraded
	default:
		return contract.Normal
	}
}

// RecordSuccess 记录成功（供服务层在评估后调用）
func (cb *CircuitBreaker) RecordSuccess() {
	switch CircuitState(cb.state.Load()) {
	case CircuitHalfOpen:
		if cb.successCount.Add(1) >= int64(cb.halfOpenRequests) {
			// CAS 确保只有一个 goroutine 执行状态转换
			if cb.state.CompareAndSwap(int32(CircuitHalfOpen), int32(CircuitClosed)) {
				cb.failureCount.Store(0)
				cb.successCount.Store(0)
			}
		}
	case CircuitClosed:
		cb.failureCount.Store(0)
	}
}

// RecordFailure 记录失败（供服务层在评估后调用）
func (cb *CircuitBreaker) RecordFailure() {
	cb.lastFailure.Store(time.Now().UnixNano())
	switch CircuitState(cb.state.Load()) {
	case CircuitHalfOpen:
		cb.state.Store(int32(CircuitOpen))
	case CircuitClosed:
		if cb.failureCount.Add(1) >= int64(cb.failureThreshold) {
			cb.state.Store(int32(CircuitOpen))
		}
	}
}

// State 返回当前状态
func (cb *CircuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}
