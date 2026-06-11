// Package contract - 背压契约
//
// V7 收敛：从 core/backpressure 迁入。
// 为避免 core/contract → core → core/trace → core/contract 的循环，
// 本包不直接引用 core.DataContext，使用 any 表示数据上下文。
// 实现方如需访问 DataContext 方法，可在内部做类型断言。
package contract

import (
	"context"
	"errors"
)

// PointIdentity 数据点身份标识（最小契约接口）。
//
// 仅声明 Tracker 在 TrackDrop/TrackError 签名上需要的方法，
// 语义上是数据点的"身份标识"，而非完整的 DataContext。
// 引擎实际的 DataContext 是 core.DataContext（更丰富），应用层若需访问完整方法，
// 可在实现 Tracker 时做类型断言。
type PointIdentity interface {
	// 基础标识（用于日志/监控）
	DeviceID() string
	PointName() string
	PointType() string

	// 数据值
	Value() float64
}

// AsPointIdentity 将 any 转换为 PointIdentity（实现方内部使用）
func AsPointIdentity(v any) (PointIdentity, bool) {
	if v == nil {
		return nil, false
	}
	pi, ok := v.(PointIdentity)
	return pi, ok
}

// MustPointIdentity 将 any 强制转换为 PointIdentity，失败时返回 nil（实现方内部使用）
func MustPointIdentity(v any) PointIdentity {
	pi, _ := AsPointIdentity(v)
	return pi
}

// ErrInvalidPointIdentity 数据点身份类型断言失败
var ErrInvalidPointIdentity = errors.New("invalid point identity type")

// Level 背压级别（应用层策略）
//
// 0=Normal 正常；1=Degraded 降级（跳过低优先级规则）；
// 2=Paused 暂停（仅 critical）；3=Dropping 丢弃（透传数据）。
type Level int

// 背压级别常量
const (
	Normal   Level = iota // 0 正常
	Degraded              // 1 降级：跳过低优先级规则
	Paused                // 2 暂停：仅执行 critical 规则
	Dropping              // 3 丢弃：透传数据
)

// Indicator 背压指示器接口。
//
// 上层实现：可基于 BackpressureManager、熔断器、限流器、信号量等任意策略。
// 引擎在评估入口调用 ShouldAccept 判定是否放行，并在循环中调用 CurrentLevel 调整策略。
type Indicator interface {
	// ShouldAccept 当前是否接收数据（设备级或全局）
	ShouldAccept(deviceID string) bool
	// CurrentLevel 当前背压级别
	CurrentLevel() Level
}

// Tracker 数据丢失追踪器接口。
//
// 上层实现：可对接 DLQ、监控系统、日志系统、Prometheus 计数器等。
// 引擎在数据被丢弃或动作执行错误时调用，**不阻塞**热路径。
//
// data 参数使用 any 而非 PointIdentity 是为了保持 Tracker 签名简洁。
// 实现方在需要时通过 AsPointIdentity(data) 做类型断言。
type Tracker interface {
	// TrackDrop 追踪数据丢弃事件
	TrackDrop(ctx context.Context, data any, ruleID string, reason string)
	// TrackError 追踪规则执行错误事件
	TrackError(ctx context.Context, data any, ruleID string, err error)
}
