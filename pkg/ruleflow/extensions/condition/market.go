// Package condition provides VPP condition nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package condition

import (
	"context"
	"fmt"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  ToUPriceCondition — 分时电价时段判断条件
// ─────────────────────────────────────────────

// touPeriod 分时电价时段定义
type touPeriod struct {
	StartHour int
	EndHour   int
}

// touSchedule 分时电价时段配置
type touSchedule struct {
	Peak         []touPeriod
	Flat         []touPeriod
	Valley       []touPeriod
	CriticalPeak []touPeriod
}

// 中国典型分时电价时段配置（示例：上海工商业夏季）
var defaultTouSchedules = map[string]map[string]*touSchedule{
	"summer": {
		"shanghai": &touSchedule{
			Peak:         []touPeriod{{8, 11}, {13, 15}, {18, 21}},
			Flat:         []touPeriod{{6, 8}, {11, 13}, {15, 18}, {21, 22}},
			Valley:       []touPeriod{{22, 24}, {0, 6}},
			CriticalPeak: []touPeriod{{8, 11}},
		},
	},
	"winter": {
		"shanghai": &touSchedule{
			Peak:   []touPeriod{{8, 11}, {13, 17}, {18, 21}},
			Flat:   []touPeriod{{6, 8}, {11, 13}, {17, 18}, {21, 22}},
			Valley: []touPeriod{{22, 24}, {0, 6}},
		},
	},
}

// ToUPriceCondition 分时电价时段判断条件
type ToUPriceCondition struct {
	IDValue  string `json:"id"`
	Period   string `json:"period"`   // peak / flat / valley / critical_peak
	Timezone string `json:"timezone"` // 时区
	Season   string `json:"season"`   // summer / winter / auto
	Region   string `json:"region"`   // 区域

	schedule *touSchedule
}

// NewToUPriceCondition 创建分时电价时段判断条件
func NewToUPriceCondition(id, period, timezone, season, region string) *ToUPriceCondition {
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	if region == "" {
		region = "shanghai"
	}

	c := &ToUPriceCondition{
		IDValue:  id,
		Period:   period,
		Timezone: timezone,
		Season:   season,
		Region:   region,
	}

	// 加载时段配置
	if seasonSchedules, ok := defaultTouSchedules[season]; ok {
		if schedule, ok := seasonSchedules[region]; ok {
			c.schedule = schedule
		}
	}

	return c
}

func (c *ToUPriceCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	if c.schedule == nil {
		return false
	}

	// 获取当前本地时间
	now := time.UnixMilli(data.Timestamp())
	loc := time.UTC
	if c.Timezone != "" {
		if l, err := time.LoadLocation(c.Timezone); err == nil {
			loc = l
		}
	}
	localNow := now.In(loc)
	hour := localNow.Hour()

	// 判断当前小时属于哪个时段
	periods := c.getPeriods()
	for _, p := range periods {
		if hour >= p.StartHour && hour < p.EndHour {
			return true
		}
	}
	return false
}

func (c *ToUPriceCondition) getPeriods() []touPeriod {
	switch c.Period {
	case "peak":
		return c.schedule.Peak
	case "flat":
		return c.schedule.Flat
	case "valley":
		return c.schedule.Valley
	case "critical_peak":
		return c.schedule.CriticalPeak
	}
	return nil
}

func (c *ToUPriceCondition) ID() string   { return c.IDValue }
func (c *ToUPriceCondition) Type() string { return "time_of_use_price" }
func (c *ToUPriceCondition) Description() string {
	return fmt.Sprintf("TOU price period=%s season=%s", c.Period, c.Season)
}

// ─────────────────────────────────────────────
//  DemandResponseCheckCondition — 需求响应执行率检测条件
// ─────────────────────────────────────────────

// DemandResponseCheckCondition 需求响应执行率检测条件
type DemandResponseCheckCondition struct {
	IDValue             string  `json:"id"`
	ExpectedReductionKW float64 `json:"expected_reduction_kw"`
	BaselinePoint       string  `json:"baseline_point"`
	ActualPoint         string  `json:"actual_point"`
	TolerancePct        float64 `json:"tolerance_pct"`
}

// NewDemandResponseCheckCondition 创建需求响应执行率检测条件
func NewDemandResponseCheckCondition(id string, expectedKW float64, baselinePoint, actualPoint string, tolerancePct float64) *DemandResponseCheckCondition {
	if tolerancePct == 0 {
		tolerancePct = 10
	}
	return &DemandResponseCheckCondition{
		IDValue:             id,
		ExpectedReductionKW: expectedKW,
		BaselinePoint:       baselinePoint,
		ActualPoint:         actualPoint,
		TolerancePct:        tolerancePct,
	}
}

func (c *DemandResponseCheckCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	mdc, ok := data.(types.MultiDataContextInterface)
	if !ok {
		return false
	}

	baseline, errB := mdc.GetPoint(c.BaselinePoint)
	actual, errA := mdc.GetPoint(c.ActualPoint)
	if errB != nil || errA != nil {
		return false
	}

	reduction := baseline - actual
	if reduction < 0 {
		return false
	}

	if c.ExpectedReductionKW <= 0 {
		return false
	}

	rate := reduction / c.ExpectedReductionKW * 100
	threshold := 100.0 - c.TolerancePct
	return rate >= threshold
}

func (c *DemandResponseCheckCondition) ID() string   { return c.IDValue }
func (c *DemandResponseCheckCondition) Type() string { return "demand_response_check" }
func (c *DemandResponseCheckCondition) Description() string {
	return fmt.Sprintf("DR check: expect %.0fkW, tolerance %.0f%%", c.ExpectedReductionKW, c.TolerancePct)
}