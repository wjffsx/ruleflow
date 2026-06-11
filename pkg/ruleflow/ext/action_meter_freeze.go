package ext

import (
	"context"
	"fmt"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  MeterFreezeAction — 电度量冻结动作
// ─────────────────────────────────────────────

// MeterStorage 电度量存储接口
type MeterStorage interface {
	SaveFreeze(deviceID, pointName string, value float64, ts int64, freezeType string) error
	GetLastFreeze(deviceID, pointName string) (float64, int64, bool)
}

// MeterFreezeAction 电度量冻结动作
// 冻结当前电度量值，用于结算等场景
type MeterFreezeAction struct {
	IDValue    string
	FreezeType string        // "instant" | "periodic"
	Period     time.Duration // 冻结周期（仅 periodic 类型）
	Storage    MeterStorage  // 外部注入
}

var _ core.Action = (*MeterFreezeAction)(nil)

// NewMeterFreezeAction 创建电度量冻结动作
func NewMeterFreezeAction(id, freezeType string, period time.Duration, storage MeterStorage) *MeterFreezeAction {
	if freezeType == "" {
		freezeType = "instant"
	}
	return &MeterFreezeAction{
		IDValue:    id,
		FreezeType: freezeType,
		Period:     period,
		Storage:    storage,
	}
}

func (a *MeterFreezeAction) Execute(_ context.Context, data core.DataContext) error {
	if a.Storage == nil {
		return nil // 无存储，跳过
	}

	deviceID := data.DeviceID()
	pointName := data.PointName()
	value := data.Value()
	ts := data.Timestamp()

	// 检查是否需要冻结
	if a.FreezeType == "periodic" && a.Period > 0 {
		// 周期冻结：检查上次冻结时间
		_, lastTs, ok := a.Storage.GetLastFreeze(deviceID, pointName)
		if ok {
			elapsed := ts - lastTs
			if elapsed < a.Period.Milliseconds() {
				// 未到冻结周期，跳过
				return nil
			}
		}
	}

	// 执行冻结
	if err := a.Storage.SaveFreeze(deviceID, pointName, value, ts, a.FreezeType); err != nil {
		return fmt.Errorf("meter freeze: %w", err)
	}

	// 输出冻结信息到 Tag
	data.SetTag("_freeze_value", fmt.Sprintf("%.6f", value))
	data.SetTag("_freeze_time", fmt.Sprintf("%d", ts))
	data.SetTag("_freeze_type", a.FreezeType)

	return nil
}

func (a *MeterFreezeAction) ID() string          { return a.IDValue }
func (a *MeterFreezeAction) Type() string        { return "meter_freeze" }
func (a *MeterFreezeAction) Description() string { return fmt.Sprintf("meter freeze %s", a.FreezeType) }