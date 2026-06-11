package ext

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
)

// ─────────────────────────────────────────────
//  DemandCalcAction — 需量计算动作
// ─────────────────────────────────────────────

// demandSample 需量采样点
type demandSample struct {
	Value     float64
	Timestamp int64
}

// DemandCalcState 需量计算状态
type DemandCalcState struct {
	Samples   []demandSample // 滑动窗口采样点
	PeakValue float64        // 周期内最大需量
	PeakTime  int64          // 需量峰值时间
}

// DemandCalcAction 需量计算动作
// 计算需量（电力系统中的需量计算）
type DemandCalcAction struct {
	IDValue    string
	Period     time.Duration // 需量周期，如 15m
	Method     string        // "sliding" | "block"
	Interval   time.Duration // 采样间隔
	StateStore core.StateStore
}

var _ core.Action = (*DemandCalcAction)(nil)

// NewDemandCalcAction 创建需量计算动作
func NewDemandCalcAction(id string, period time.Duration, method string, interval time.Duration, stateStore core.StateStore) *DemandCalcAction {
	if method == "" {
		method = "sliding"
	}
	if interval == 0 {
		interval = time.Second
	}
	return &DemandCalcAction{
		IDValue:    id,
		Period:     period,
		Method:     method,
		Interval:   interval,
		StateStore: stateStore,
	}
}

func (a *DemandCalcAction) Execute(_ context.Context, data core.DataContext) error {
	if a.StateStore == nil {
		return nil // 无状态存储，跳过
	}

	key := util.StateKey("demand_calc", a.IDValue, data.DeviceID(), data.PointName())
	nowTs := data.Timestamp()
	val := data.Value()

	// 加载或初始化状态
	stateI, loaded := a.StateStore.Get(key)
	var state *DemandCalcState
	if loaded {
		if s, ok := stateI.(*DemandCalcState); ok {
			state = s
		}
	}
	if state == nil {
		state = &DemandCalcState{}
	}

	// 追加采样点
	state.Samples = append(state.Samples, demandSample{
		Value:     val,
		Timestamp: nowTs,
	})

	// 淘汰过期采样点
	cutoff := nowTs - a.Period.Milliseconds()
	i := 0
	for i < len(state.Samples) && state.Samples[i].Timestamp < cutoff {
		i++
	}
	if i > 0 {
		state.Samples = state.Samples[i:]
	}

	// 计算需量
	var demandValue float64
	if len(state.Samples) > 0 {
		switch a.Method {
		case "sliding":
			// 滑动窗口法：计算窗口内平均值
			var sum float64
			for _, s := range state.Samples {
				sum += s.Value
			}
			demandValue = sum / float64(len(state.Samples))
		case "block":
			// 固定周期法：计算周期内最大值
			for _, s := range state.Samples {
				if s.Value > demandValue {
					demandValue = s.Value
				}
			}
		default:
			demandValue = val
		}
	}

	// 更新峰值
	if demandValue > state.PeakValue {
		state.PeakValue = demandValue
		state.PeakTime = nowTs
	}

	// 保存状态
	a.StateStore.Set(key, state)

	// 输出需量信息到 Tag
	data.SetTag("_demand_value", fmt.Sprintf("%.6f", demandValue))
	data.SetTag("_demand_peak", fmt.Sprintf("%.6f", state.PeakValue))
	data.SetTag("_demand_time", strconv.FormatInt(state.PeakTime, 10))
	data.SetTag("_demand_method", a.Method)
	data.SetTag("_demand_samples", strconv.Itoa(len(state.Samples)))

	return nil
}

func (a *DemandCalcAction) ID() string          { return a.IDValue }
func (a *DemandCalcAction) Type() string        { return "demand_calc" }
func (a *DemandCalcAction) Description() string { return fmt.Sprintf("demand calc %s over %v", a.Method, a.Period) }