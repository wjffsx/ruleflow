package ext

import (
	"context"
	"strconv"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
)

// ─────────────────────────────────────────────
//  LimitTrackerAction — 越限状态跟踪动作
// ─────────────────────────────────────────────

// LimitTrackerState 越限状态跟踪状态
type LimitTrackerState struct {
	ExceededCount    int   // 越限次数
	ExceededDuration int64 // 越限累计持续时间（毫秒）
	LastExceededTime int64 // 最近越限时间
	RecoveryTime     int64 // 最近恢复时间
	IsExceeded       bool  // 当前是否越限
}

// LimitTrackerAction 越限状态跟踪动作
// 跟踪越限次数、持续时间等统计信息
type LimitTrackerAction struct {
	IDValue       string
	TrackDuration bool    // 是否跟踪持续时间
	TrackCount    bool    // 是否跟踪越限次数
	Hysteresis    float64 // 迟滞值
	StateStore    core.StateStore // 外部注入
}

var _ core.Action = (*LimitTrackerAction)(nil)

// NewLimitTrackerAction 创建越限状态跟踪动作
func NewLimitTrackerAction(id string, trackDuration, trackCount bool, hysteresis float64, stateStore core.StateStore) *LimitTrackerAction {
	return &LimitTrackerAction{
		IDValue:       id,
		TrackDuration: trackDuration,
		TrackCount:    trackCount,
		Hysteresis:    hysteresis,
		StateStore:    stateStore,
	}
}

func (a *LimitTrackerAction) Execute(_ context.Context, data core.DataContext) error {
	if a.StateStore == nil {
		return nil // 无状态存储，跳过
	}

	key := util.StateKey("limit_tracker", a.IDValue, data.DeviceID(), data.PointName())
	nowTs := data.Timestamp()
	val := data.Value()

	// 获取上下限
	upper, hasUpper := data.UpperLimit()
	lower, hasLower := data.LowerLimit()

	// 判断当前是否越限（考虑迟滞）
	isExceeded := false
	if hasUpper && val > upper {
		isExceeded = true
	}
	if hasLower && val < lower {
		isExceeded = true
	}

	// 加载或初始化状态
	stateI, loaded := a.StateStore.Get(key)
	var state *LimitTrackerState
	if loaded {
		if s, ok := stateI.(*LimitTrackerState); ok {
			state = s
		}
	}
	if state == nil {
		state = &LimitTrackerState{}
	}

	// 更新状态
	if isExceeded {
		if !state.IsExceeded {
			// 从正常变为越限
			if a.TrackCount {
				state.ExceededCount++
			}
			state.LastExceededTime = nowTs
			state.IsExceeded = true
		} else {
			// 持续越限
			if a.TrackDuration && state.LastExceededTime > 0 {
				state.ExceededDuration += nowTs - state.LastExceededTime
				state.LastExceededTime = nowTs
			}
		}
	} else {
		if state.IsExceeded {
			// 从越限恢复
			state.RecoveryTime = nowTs
			state.IsExceeded = false
		}
	}

	// 保存状态
	a.StateStore.Set(key, state)

	// 输出统计信息到 Tag
	data.SetTag("_limit_exceeded_count", strconv.Itoa(state.ExceededCount))
	data.SetTag("_limit_exceeded_duration", strconv.FormatInt(state.ExceededDuration, 10))
	data.SetTag("_limit_recovery_time", strconv.FormatInt(state.RecoveryTime, 10))
	data.SetTag("_limit_is_exceeded", strconv.FormatBool(state.IsExceeded))

	return nil
}

func (a *LimitTrackerAction) ID() string          { return a.IDValue }
func (a *LimitTrackerAction) Type() string        { return "limit_tracker" }
func (a *LimitTrackerAction) Description() string { return "limit tracker" }