// Package contract - 优雅关闭状态枚举
package contract

// ShutdownState 关闭状态
type ShutdownState int32

const (
	// ShutdownStateRunning 正常运行
	ShutdownStateRunning ShutdownState = iota
	// ShutdownStateShuttingDown 正在关闭（拒绝新评估）
	ShutdownStateShuttingDown
	// ShutdownStateShutdown 已关闭
	ShutdownStateShutdown
)

// String 返回关闭状态字符串
func (s ShutdownState) String() string {
	switch s {
	case ShutdownStateRunning:
		return "running"
	case ShutdownStateShuttingDown:
		return "shutting_down"
	case ShutdownStateShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}
