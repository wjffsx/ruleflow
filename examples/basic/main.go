package main

import (
	"context"
	"fmt"
	"log"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/action"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/engine"
)

// simpleDataPoint 简单的 DataContext 实现，用于示例
type simpleDataPoint struct {
	deviceID      string
	pointName     string
	pointType     string
	value         float64
	quality       int
	limitExceeded bool
	dropped       bool
	targets       []string
	tags          map[string]string
}

func (d *simpleDataPoint) DeviceID() string                      { return d.deviceID }
func (d *simpleDataPoint) PointName() string                     { return d.pointName }
func (d *simpleDataPoint) SetPointName(name string)              { d.pointName = name }
func (d *simpleDataPoint) PointType() string                     { return d.pointType }
func (d *simpleDataPoint) FQN() string                           { return d.deviceID + "/" + d.pointName }
func (d *simpleDataPoint) Value() float64                        { return d.value }
func (d *simpleDataPoint) SetValue(v float64)                    { d.value = v }
func (d *simpleDataPoint) Quality() int                          { return d.quality }
func (d *simpleDataPoint) SetQuality(q int)                      { d.quality = q }
func (d *simpleDataPoint) UpperLimit() (float64, bool)           { return 0, false }
func (d *simpleDataPoint) LowerLimit() (float64, bool)           { return 0, false }
func (d *simpleDataPoint) LimitExceeded() bool                   { return d.limitExceeded }
func (d *simpleDataPoint) SetLimitExceeded(v bool)               { d.limitExceeded = v }
func (d *simpleDataPoint) GetTag(key string) string              { return d.tags[key] }
func (d *simpleDataPoint) SetTag(key, value string)              { d.tags[key] = value }
func (d *simpleDataPoint) TargetCount() int                      { return len(d.targets) }
func (d *simpleDataPoint) TargetAt(i int) string                 { return d.targets[i] }
func (d *simpleDataPoint) AddTarget(target string)               { d.targets = append(d.targets, target) }
func (d *simpleDataPoint) Dropped() bool                         { return d.dropped }
func (d *simpleDataPoint) SetDropped(v bool)                     { d.dropped = v }
func (d *simpleDataPoint) Timestamp() int64                      { return 0 }
func (d *simpleDataPoint) SpanContext() contract.SpanContext     { return contract.SpanContext{} }
func (d *simpleDataPoint) SetSpanContext(_ contract.SpanContext) {}
func (d *simpleDataPoint) Raw() any                              { return d }
func (d *simpleDataPoint) PreviousValue() (float64, bool)        { return 0, false }
func (d *simpleDataPoint) SetPreviousValue(float64)              {}

func newDataPoint(deviceType, pointName string, value float64) *simpleDataPoint {
	return &simpleDataPoint{
		pointType: deviceType,
		pointName: pointName,
		value:     value,
		tags:      make(map[string]string),
	}
}

func main() {
	// 1. 创建引擎
	eng := engine.NewEngine()

	// 2. 构建规则链
	scale := 0.1
	chain := &core.RuleChain{
		ID:      "demo_chain",
		Name:    "示例规则链",
		Root:    true,
		Version: 1,
		Status:  "deployed",
		Rules: []*core.Rule{
			{
				ID:       "filter_analog",
				Name:     "过滤模拟量 + 值变换",
				Priority: 1,
				Enabled:  true,
				Condition: &core.ConditionNode{
					Leaf: condition.NewDeviceTypeCondition("c1", []string{"analog"}),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						action.NewTransformAction("a1", &scale, nil, "kV"),
						action.NewLimitCheckAction("a2"),
						action.NewQualityMarkAction("a3", 0, 1),
					},
				},
				Targets: []string{"default"},
			},
			{
				ID:       "drop_digital",
				Name:     "丢弃数字量",
				Priority: 2,
				Enabled:  true,
				Condition: &core.ConditionNode{
					Leaf: condition.NewDeviceTypeCondition("c2", []string{"digital"}),
				},
				Actions: &core.ActionChain{
					Actions: []core.Action{
						action.NewDropAction("a4"),
					},
				},
			},
		},
	}

	// 3. 加载并编译
	if err := eng.LoadChain(chain); err != nil {
		log.Fatalf("load chain: %v", err)
	}

	// 4. 评估模拟量数据点
	result, err := eng.EvalChain(context.Background(), "demo_chain", newDataPoint("analog", "voltage", 220.5))
	if err != nil {
		log.Fatalf("eval: %v", err)
	}
	fmt.Printf("模拟量: matched=%d, dropped=%v, targets=%v\n",
		len(result.MatchedRules), result.Dropped, result.Data.(*simpleDataPoint).targets)

	// 5. 评估数字量数据点（应被丢弃）
	result2, err := eng.EvalChain(context.Background(), "demo_chain", newDataPoint("digital", "switch", 1.0))
	if err != nil {
		log.Fatalf("eval: %v", err)
	}
	fmt.Printf("数字量: matched=%d, dropped=%v\n",
		len(result2.MatchedRules), result2.Dropped)

	fmt.Println("ruleflow basic example completed!")
}
