// Package contract - 指标契约
package contract

import "time"

// MetricsSink 引擎指标导出接口（可插拔）。
type MetricsSink interface {
	// 评估级
	IncEvalTotal(chainID, result string)
	ObserveEvalDuration(chainID string, d time.Duration)

	// 节点级
	ObserveConditionEval(chainID, nodeID string, duration time.Duration, matched bool)
	ObserveActionExec(chainID, nodeID, actionType string, duration time.Duration, hasError bool)
	IncActionTotal(chainID, actionType, result string)

	// 异常
	IncRuleThrottled(chainID, ruleID string)
	IncPanic(chainID, ruleID string)

	// 引擎元数据
	SetLoadedChains(n int)
	SetActiveEval(n int64)
}

// NoopSink 返回零开销的 MetricsSink。
func NoopSink() MetricsSink { return noopSink{} }

type noopSink struct{}

func (noopSink) IncEvalTotal(_, _ string)                                  {}
func (noopSink) ObserveEvalDuration(_ string, _ time.Duration)             {}
func (noopSink) ObserveConditionEval(_, _ string, _ time.Duration, _ bool) {}
func (noopSink) ObserveActionExec(_, _, _ string, _ time.Duration, _ bool) {}
func (noopSink) IncActionTotal(_, _, _ string)                             {}
func (noopSink) IncRuleThrottled(_, _ string)                              {}
func (noopSink) IncPanic(_, _ string)                                      {}
func (noopSink) SetLoadedChains(_ int)                                     {}
func (noopSink) SetActiveEval(_ int64)                                     {}

// 编译期接口检查
var _ MetricsSink = noopSink{}
