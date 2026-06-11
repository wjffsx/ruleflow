// Package prometheus 提供 ruleflow 引擎指标的 Prometheus 导出实现。
//
// 该包实现了 contract.MetricsSink 接口，
// 使用 promauto 自动注册到 Prometheus 默认 registry，用户无需手动注册。
//
// 基本用法:
//
//	import "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/prometheus"
//
//	sink := prometheus.NewPrometheusSink()
//	engine := ruleflow.NewEngine(ruleflow.WithMetricsSink(sink))
//
//	// 暴露 HTTP 端点
//	http.Handle("/metrics", sink.Handler())
//	log.Fatal(http.ListenAndServe(":9090", nil))
package prometheus

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  Prometheus 指标定义
// ─────────────────────────────────────────────

// evalDurationBuckets 定义链评估耗时的 Histogram Buckets（秒）。
// 覆盖微妙到毫秒范围，适合规则引擎的典型延迟分布。
var evalDurationBuckets = []float64{
	0.000001, // 1 μs
	0.000005, // 5 μs
	0.000010, // 10 μs
	0.000050, // 50 μs
	0.000100, // 100 μs
	0.000500, // 500 μs
	0.001,    // 1 ms
	0.005,    // 5 ms
	0.010,    // 10 ms
}

// metrics 名称常量
const (
	metricEvalTotal          = "ruleflow_eval_total"
	metricActionTotal        = "ruleflow_action_total"
	metricRuleThrottled      = "ruleflow_rule_throttled_total"
	metricPanicTotal         = "ruleflow_panic_total"
	metricEvalDuration       = "ruleflow_eval_duration_seconds"
	metricLoadedChains       = "ruleflow_loaded_chains"
	metricActiveEval         = "ruleflow_active_eval"
	metricConditionEvalTotal = "ruleflow_condition_eval_total"
	metricConditionEvalDur   = "ruleflow_condition_eval_duration_seconds"
	metricActionExecTotal    = "ruleflow_action_exec_total"
	metricActionExecDur      = "ruleflow_action_exec_duration_seconds"
)

// ─────────────────────────────────────────────
//  PrometheusSink — 实现 metrics.MetricsSink
// ─────────────────────────────────────────────

// PrometheusSink 实现 metrics.MetricsSink 接口，将引擎指标暴露为 Prometheus 格式。
// 所有指标通过 promauto 自动注册到 prometheus.DefaultRegisterer。
type PrometheusSink struct {
	// Counter 指标
	evalTotal          *prometheus.CounterVec
	actionTotal        *prometheus.CounterVec
	ruleThrottled      *prometheus.CounterVec
	panicTotal         *prometheus.CounterVec
	conditionEvalTotal *prometheus.CounterVec
	actionExecTotal    *prometheus.CounterVec

	// Histogram 指标
	evalDuration     *prometheus.HistogramVec
	conditionEvalDur *prometheus.HistogramVec
	actionExecDur    *prometheus.HistogramVec

	// Gauge 指标
	loadedChains prometheus.Gauge
	activeEval   prometheus.Gauge
}

// NewPrometheusSink 创建 PrometheusSink。
// 所有指标自动注册到 Prometheus 默认 registry，无需额外注册步骤。
func NewPrometheusSink() *PrometheusSink {
	return &PrometheusSink{
		evalTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricEvalTotal,
				Help: "Total number of rule chain evaluations, partitioned by chain and result.",
			},
			[]string{"chain_id", "result"},
		),
		actionTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricActionTotal,
				Help: "Total number of action executions, partitioned by chain, action type and result.",
			},
			[]string{"chain_id", "action_type", "result"},
		),
		ruleThrottled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricRuleThrottled,
				Help: "Total number of rules throttled (skipped due to rate limiting), partitioned by chain and rule.",
			},
			[]string{"chain_id", "rule_id"},
		),
		panicTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricPanicTotal,
				Help: "Total number of panics recovered during rule evaluation, partitioned by chain and rule.",
			},
			[]string{"chain_id", "rule_id"},
		),
		conditionEvalTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricConditionEvalTotal,
				Help: "Total number of condition node evaluations, partitioned by chain and node.",
			},
			[]string{"chain_id", "node_id", "matched"},
		),
		actionExecTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricActionExecTotal,
				Help: "Total number of action node executions, partitioned by chain, node and error status.",
			},
			[]string{"chain_id", "node_id", "action_type", "has_error"},
		),
		evalDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metricEvalDuration,
				Help:    "Duration of rule chain evaluations in seconds.",
				Buckets: evalDurationBuckets,
			},
			[]string{"chain_id"},
		),
		conditionEvalDur: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metricConditionEvalDur,
				Help:    "Duration of condition node evaluations in seconds.",
				Buckets: evalDurationBuckets,
			},
			[]string{"chain_id", "node_id"},
		),
		actionExecDur: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metricActionExecDur,
				Help:    "Duration of action node executions in seconds.",
				Buckets: evalDurationBuckets,
			},
			[]string{"chain_id", "node_id", "action_type"},
		),
		loadedChains: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: metricLoadedChains,
				Help: "Current number of loaded rule chains.",
			},
		),
		activeEval: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: metricActiveEval,
				Help: "Current number of active (in-flight) rule evaluations.",
			},
		),
	}
}

// Handler 返回 promhttp.Handler，可直接挂载到 HTTP 端点供 Prometheus 抓取。
func (s *PrometheusSink) Handler() http.Handler {
	return promhttp.Handler()
}

// IncEvalTotal 实现 metrics.MetricsSink。
func (s *PrometheusSink) IncEvalTotal(chainID, result string) {
	s.evalTotal.WithLabelValues(chainID, result).Inc()
}

// IncActionTotal 实现 metrics.MetricsSink。
func (s *PrometheusSink) IncActionTotal(chainID, actionType, result string) {
	s.actionTotal.WithLabelValues(chainID, actionType, result).Inc()
}

// IncRuleThrottled 实现 metrics.MetricsSink。
func (s *PrometheusSink) IncRuleThrottled(chainID, ruleID string) {
	s.ruleThrottled.WithLabelValues(chainID, ruleID).Inc()
}

// IncPanic 实现 metrics.MetricsSink。
func (s *PrometheusSink) IncPanic(chainID, ruleID string) {
	s.panicTotal.WithLabelValues(chainID, ruleID).Inc()
}

// ObserveEvalDuration 实现 metrics.MetricsSink。
func (s *PrometheusSink) ObserveEvalDuration(chainID string, d time.Duration) {
	s.evalDuration.WithLabelValues(chainID).Observe(d.Seconds())
}

// SetLoadedChains 实现 metrics.MetricsSink。
func (s *PrometheusSink) SetLoadedChains(n int) {
	s.loadedChains.Set(float64(n))
}

// SetActiveEval 实现 metrics.MetricsSink。
func (s *PrometheusSink) SetActiveEval(n int64) {
	s.activeEval.Set(float64(n))
}

// ObserveConditionEval 实现 metrics.MetricsSink。
func (s *PrometheusSink) ObserveConditionEval(chainID, nodeID string, duration time.Duration, matched bool) {
	matchedStr := "false"
	if matched {
		matchedStr = "true"
	}
	s.conditionEvalTotal.WithLabelValues(chainID, nodeID, matchedStr).Inc()
	s.conditionEvalDur.WithLabelValues(chainID, nodeID).Observe(duration.Seconds())
}

// ObserveActionExec 实现 metrics.MetricsSink。
func (s *PrometheusSink) ObserveActionExec(chainID, nodeID, actionType string, duration time.Duration, hasError bool) {
	hasErrStr := "false"
	if hasError {
		hasErrStr = "true"
	}
	s.actionExecTotal.WithLabelValues(chainID, nodeID, actionType, hasErrStr).Inc()
	s.actionExecDur.WithLabelValues(chainID, nodeID, actionType).Observe(duration.Seconds())
}

// ─────────────────────────────────────────────
//  编译期接口检查
// ─────────────────────────────────────────────

var _ contract.MetricsSink = (*PrometheusSink)(nil)
