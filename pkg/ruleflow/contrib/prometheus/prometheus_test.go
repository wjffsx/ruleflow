package prometheus

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// ─────────────────────────────────────────────
//  TestMain — 一次性创建 Sink
//  promauto 注册到 default registry，多次调用会 panic，
//  因此通过 TestMain 全局创建一次，所有测试共享。
// ─────────────────────────────────────────────

var globalSink *PrometheusSink

func TestMain(m *testing.M) {
	globalSink = NewPrometheusSink()
	os.Exit(m.Run())
}

// ─────────────────────────────────────────────
//  PrometheusSink 基础测试
// ─────────────────────────────────────────────

func TestNewPrometheusSink_NonNil(t *testing.T) {
	if globalSink == nil {
		t.Fatal("NewPrometheusSink() returned nil")
	}
}

func TestPrometheusSink_Handler_NonNil(t *testing.T) {
	h := globalSink.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil")
	}
	if _, ok := h.(http.Handler); !ok {
		t.Fatal("Handler() does not implement http.Handler")
	}
}

func TestPrometheusSink_ImplementsMetricsSink(t *testing.T) {
	var sink contract.MetricsSink = globalSink
	if sink == nil {
		t.Fatal("PrometheusSink does not implement MetricsSink")
	}
}

func TestPrometheusSink_Methods_NoPanic(t *testing.T) {
	globalSink.IncEvalTotal("test-chain", "matched")
	globalSink.IncActionTotal("test-chain", "rule", "ok")
	globalSink.IncRuleThrottled("test-chain", "rule-1")
	globalSink.IncPanic("test-chain", "rule-1")
	globalSink.ObserveEvalDuration("test-chain", 100*time.Microsecond)
	globalSink.SetLoadedChains(5)
	globalSink.SetActiveEval(3)
	globalSink.ObserveConditionEval("test-chain", "cond-1", 50*time.Microsecond, true)
	globalSink.ObserveActionExec("test-chain", "act-1", "http", 200*time.Microsecond, false)
}

// ─────────────────────────────────────────────
//  PrometheusSink 功能验证（使用独立 registry）
// ─────────────────────────────────────────────

func TestPrometheusSink_EvalTotal_Counter(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.IncEvalTotal("chain-a", "matched")
	sink.IncEvalTotal("chain-a", "matched")
	sink.IncEvalTotal("chain-a", "unmatched")

	// 验证计数
	want := `# HELP ruleflow_eval_total Total number of rule chain evaluations, partitioned by chain and result.
# TYPE ruleflow_eval_total counter
ruleflow_eval_total{chain_id="chain-a",result="matched"} 2
ruleflow_eval_total{chain_id="chain-a",result="unmatched"} 1
`
	if err := testutil.CollectAndCompare(sink.evalTotal, stringToReader(want), metricEvalTotal); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_ActionTotal_Counter(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.IncActionTotal("chain-a", "http", "ok")
	sink.IncActionTotal("chain-a", "http", "errored")

	want := `# HELP ruleflow_action_total Total number of action executions, partitioned by chain, action type and result.
# TYPE ruleflow_action_total counter
ruleflow_action_total{action_type="http",chain_id="chain-a",result="errored"} 1
ruleflow_action_total{action_type="http",chain_id="chain-a",result="ok"} 1
`
	if err := testutil.CollectAndCompare(sink.actionTotal, stringToReader(want), metricActionTotal); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_RuleThrottled_Counter(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.IncRuleThrottled("chain-a", "rule-1")

	want := `# HELP ruleflow_rule_throttled_total Total number of rules throttled (skipped due to rate limiting), partitioned by chain and rule.
# TYPE ruleflow_rule_throttled_total counter
ruleflow_rule_throttled_total{chain_id="chain-a",rule_id="rule-1"} 1
`
	if err := testutil.CollectAndCompare(sink.ruleThrottled, stringToReader(want), metricRuleThrottled); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_PanicTotal_Counter(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.IncPanic("chain-a", "rule-1")
	sink.IncPanic("chain-a", "rule-1")

	want := `# HELP ruleflow_panic_total Total number of panics recovered during rule evaluation, partitioned by chain and rule.
# TYPE ruleflow_panic_total counter
ruleflow_panic_total{chain_id="chain-a",rule_id="rule-1"} 2
`
	if err := testutil.CollectAndCompare(sink.panicTotal, stringToReader(want), metricPanicTotal); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_EvalDuration_Histogram(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.ObserveEvalDuration("chain-b", 1*time.Microsecond)
	sink.ObserveEvalDuration("chain-b", 100*time.Microsecond)
	sink.ObserveEvalDuration("chain-b", 5*time.Millisecond)

	count, err := testutil.GatherAndCount(reg)
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least 1 metric family")
	}

	// 验证直方图有观测值
	c := testutil.CollectAndCount(sink.evalDuration)
	if c == 0 {
		t.Fatal("expected histogram samples")
	}
}

func TestPrometheusSink_LoadedChains_Gauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.SetLoadedChains(10)
	want := `# HELP ruleflow_loaded_chains Current number of loaded rule chains.
# TYPE ruleflow_loaded_chains gauge
ruleflow_loaded_chains 10
`
	if err := testutil.CollectAndCompare(sink.loadedChains, stringToReader(want), metricLoadedChains); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_ActiveEval_Gauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.SetActiveEval(7)
	want := `# HELP ruleflow_active_eval Current number of active (in-flight) rule evaluations.
# TYPE ruleflow_active_eval gauge
ruleflow_active_eval 7
`
	if err := testutil.CollectAndCompare(sink.activeEval, stringToReader(want), metricActiveEval); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_ObserveConditionEval(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.ObserveConditionEval("chain-c", "cond-x", 30*time.Microsecond, true)
	sink.ObserveConditionEval("chain-c", "cond-x", 40*time.Microsecond, false)
	sink.ObserveConditionEval("chain-c", "cond-y", 50*time.Microsecond, true)

	count, err := testutil.GatherAndCount(reg)
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least 1 metric family")
	}

	// 验证 condition eval 计数
	wantCounter := `# HELP ruleflow_condition_eval_total Total number of condition node evaluations, partitioned by chain and node.
# TYPE ruleflow_condition_eval_total counter
ruleflow_condition_eval_total{chain_id="chain-c",matched="false",node_id="cond-x"} 1
ruleflow_condition_eval_total{chain_id="chain-c",matched="true",node_id="cond-x"} 1
ruleflow_condition_eval_total{chain_id="chain-c",matched="true",node_id="cond-y"} 1
`
	if err := testutil.CollectAndCompare(sink.conditionEvalTotal, stringToReader(wantCounter), metricConditionEvalTotal); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

func TestPrometheusSink_ObserveActionExec(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := newSinkWithRegistry(reg)

	sink.ObserveActionExec("chain-d", "act-x", "http", 100*time.Microsecond, false)
	sink.ObserveActionExec("chain-d", "act-x", "http", 200*time.Microsecond, true)

	count, err := testutil.GatherAndCount(reg)
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least 1 metric family")
	}

	wantCounter := `# HELP ruleflow_action_exec_total Total number of action node executions, partitioned by chain, node and error status.
# TYPE ruleflow_action_exec_total counter
ruleflow_action_exec_total{action_type="http",chain_id="chain-d",has_error="false",node_id="act-x"} 1
ruleflow_action_exec_total{action_type="http",chain_id="chain-d",has_error="true",node_id="act-x"} 1
`
	if err := testutil.CollectAndCompare(sink.actionExecTotal, stringToReader(wantCounter), metricActionExecTotal); err != nil {
		t.Fatalf("unexpected metric output: %v", err)
	}
}

// ─────────────────────────────────────────────
//  Helpers
// ─────────────────────────────────────────────

// newSinkWithRegistry 创建一个使用独立 registry 的 PrometheusSink，
// 避免测试间相互干扰。
func newSinkWithRegistry(reg prometheus.Registerer) *PrometheusSink {
	sink := &PrometheusSink{
		evalTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricEvalTotal,
				Help: "Total number of rule chain evaluations, partitioned by chain and result.",
			},
			[]string{"chain_id", "result"},
		),
		actionTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricActionTotal,
				Help: "Total number of action executions, partitioned by chain, action type and result.",
			},
			[]string{"chain_id", "action_type", "result"},
		),
		ruleThrottled: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricRuleThrottled,
				Help: "Total number of rules throttled (skipped due to rate limiting), partitioned by chain and rule.",
			},
			[]string{"chain_id", "rule_id"},
		),
		panicTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricPanicTotal,
				Help: "Total number of panics recovered during rule evaluation, partitioned by chain and rule.",
			},
			[]string{"chain_id", "rule_id"},
		),
		conditionEvalTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricConditionEvalTotal,
				Help: "Total number of condition node evaluations, partitioned by chain and node.",
			},
			[]string{"chain_id", "node_id", "matched"},
		),
		actionExecTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricActionExecTotal,
				Help: "Total number of action node executions, partitioned by chain, node and error status.",
			},
			[]string{"chain_id", "node_id", "action_type", "has_error"},
		),
		evalDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metricEvalDuration,
				Help:    "Duration of rule chain evaluations in seconds.",
				Buckets: evalDurationBuckets,
			},
			[]string{"chain_id"},
		),
		conditionEvalDur: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metricConditionEvalDur,
				Help:    "Duration of condition node evaluations in seconds.",
				Buckets: evalDurationBuckets,
			},
			[]string{"chain_id", "node_id"},
		),
		actionExecDur: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metricActionExecDur,
				Help:    "Duration of action node executions in seconds.",
				Buckets: evalDurationBuckets,
			},
			[]string{"chain_id", "node_id", "action_type"},
		),
		loadedChains: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: metricLoadedChains,
				Help: "Current number of loaded rule chains.",
			},
		),
		activeEval: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: metricActiveEval,
				Help: "Current number of active (in-flight) rule evaluations.",
			},
		),
	}
	reg.MustRegister(
		sink.evalTotal,
		sink.actionTotal,
		sink.ruleThrottled,
		sink.panicTotal,
		sink.conditionEvalTotal,
		sink.actionExecTotal,
		sink.evalDuration,
		sink.conditionEvalDur,
		sink.actionExecDur,
		sink.loadedChains,
		sink.activeEval,
	)
	return sink
}

// stringToReader 将字符串转换为 io.Reader，供 testutil.CollectAndCompare 使用。
func stringToReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
