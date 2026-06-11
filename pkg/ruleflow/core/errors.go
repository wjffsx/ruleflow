package core

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

// ─────────────────────────────────────────────
//  哨兵错误（用于 errors.Is 判定）
// ─────────────────────────────────────────────

var (
	// ErrConfigInvalid 配置无效
	ErrConfigInvalid = errors.New("config invalid")
	// ErrConditionEvalFailed 条件评估失败
	ErrConditionEvalFailed = errors.New("condition evaluation failed")
	// ErrActionExecFailed 动作执行失败
	ErrActionExecFailed = errors.New("action execution failed")
	// ErrChainNotFound 规则链未找到
	ErrChainNotFound = errors.New("chain not found")
	// ErrEngineShutdown 引擎已关闭
	ErrEngineShutdown = errors.New("engine is shutdown")
	// ErrCyclicDependency 循环依赖
	ErrCyclicDependency = errors.New("cyclic dependency detected")
	// ErrPanicRecovered panic 已恢复
	ErrPanicRecovered = errors.New("panic recovered")
)

// ErrorType 错误类型分类
type ErrorType int

const (
	// ErrorTypeConfig 配置类错误
	ErrorTypeConfig ErrorType = iota
	// ErrorTypeCondition 条件评估类错误
	ErrorTypeCondition
	// ErrorTypeAction 动作执行类错误
	ErrorTypeAction
	// ErrorTypeTimeout 超时类错误
	ErrorTypeTimeout
	// ErrorTypePanic panic 恢复类错误
	ErrorTypePanic
	// ErrorTypeDependency 依赖关系类错误
	ErrorTypeDependency
	// ErrorTypeResource 资源类错误
	ErrorTypeResource
	// ErrorTypeShutdown 关闭类错误
	ErrorTypeShutdown
)

// String 返回错误类型的字符串表示
func (t ErrorType) String() string {
	switch t {
	case ErrorTypeConfig:
		return "config"
	case ErrorTypeCondition:
		return "condition"
	case ErrorTypeAction:
		return "action"
	case ErrorTypeTimeout:
		return "timeout"
	case ErrorTypePanic:
		return "panic"
	case ErrorTypeDependency:
		return "dependency"
	case ErrorTypeResource:
		return "resource"
	case ErrorTypeShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

// ─────────────────────────────────────────────
//  RuleFlowError 结构化错误
// ─────────────────────────────────────────────

// RuleFlowError 规则引擎的结构化错误。
// 包含错误类型、链/规则/动作 ID、原因、栈追踪等上下文信息。
type RuleFlowError struct {
	// Type 错误类型
	Type ErrorType
	// ChainID 关联的规则链 ID
	ChainID string
	// RuleID 关联的规则 ID
	RuleID string
	// ActionID 关联的动作 ID
	ActionID string
	// Cause 原始错误
	Cause error
	// Retryable 是否可重试
	Retryable bool
	// Stack 栈追踪（仅 panic 类型填充）
	Stack string
	// Timestamp 错误发生时间
	Timestamp time.Time
}

// Error 实现 error 接口
func (e *RuleFlowError) Error() string {
	parts := fmt.Sprintf("[%s]", e.Type)
	if e.ChainID != "" {
		parts += fmt.Sprintf(" chain=%s", e.ChainID)
	}
	if e.RuleID != "" {
		parts += fmt.Sprintf(" rule=%s", e.RuleID)
	}
	if e.ActionID != "" {
		parts += fmt.Sprintf(" action=%s", e.ActionID)
	}
	if e.Cause != nil {
		parts += fmt.Sprintf(": %v", e.Cause)
	}
	return parts
}

// Unwrap 支持 errors.Unwrap
func (e *RuleFlowError) Unwrap() error {
	return e.Cause
}

// Is 支持 errors.Is，与哨兵错误比较
func (e *RuleFlowError) Is(target error) bool {
	switch target {
	case ErrConfigInvalid:
		return e.Type == ErrorTypeConfig
	case ErrConditionEvalFailed:
		return e.Type == ErrorTypeCondition
	case ErrActionExecFailed:
		return e.Type == ErrorTypeAction
	case ErrCyclicDependency:
		return e.Type == ErrorTypeDependency
	case ErrPanicRecovered:
		return e.Type == ErrorTypePanic
	case ErrEngineShutdown:
		return e.Type == ErrorTypeShutdown
	case context.DeadlineExceeded:
		return e.Type == ErrorTypeTimeout
	}
	return false
}

// ─────────────────────────────────────────────
//  错误构造函数
// ─────────────────────────────────────────────

// NewConfigError 创建配置错误
func NewConfigError(chainID string, cause error) *RuleFlowError {
	return &RuleFlowError{
		Type:      ErrorTypeConfig,
		ChainID:   chainID,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewConditionError 创建条件评估错误
func NewConditionError(chainID, ruleID string, cause error) *RuleFlowError {
	return &RuleFlowError{
		Type:      ErrorTypeCondition,
		ChainID:   chainID,
		RuleID:    ruleID,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewActionError 创建动作执行错误
func NewActionError(chainID, ruleID, actionID string, cause error) *RuleFlowError {
	return &RuleFlowError{
		Type:      ErrorTypeAction,
		ChainID:   chainID,
		RuleID:    ruleID,
		ActionID:  actionID,
		Cause:     cause,
		Retryable: isRetryable(cause),
		Timestamp: time.Now(),
	}
}

// NewPanicError 创建 panic 恢复错误
func NewPanicError(chainID, ruleID string, r any) *RuleFlowError {
	cause := fmt.Errorf("panic: %v", r)
	return &RuleFlowError{
		Type:      ErrorTypePanic,
		ChainID:   chainID,
		RuleID:    ruleID,
		Cause:     cause,
		Stack:     string(debug.Stack()),
		Timestamp: time.Now(),
	}
}

// NewDependencyError 创建依赖错误
func NewDependencyError(chainID string, cycle []string) *RuleFlowError {
	cause := fmt.Errorf("%w: %v", ErrCyclicDependency, cycle)
	return &RuleFlowError{
		Type:      ErrorTypeDependency,
		ChainID:   chainID,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewShutdownError 创建关闭错误
func NewShutdownError(cause error) *RuleFlowError {
	return &RuleFlowError{
		Type:      ErrorTypeShutdown,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// isRetryable 判断错误是否可重试（基于错误类型启发式）
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// context.DeadlineExceeded 可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return false
}
