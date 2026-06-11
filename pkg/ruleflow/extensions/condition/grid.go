// Package condition provides VPP condition nodes
//
// V7 Refactoring: Updated to use vpp/types instead of core/vpp.
package condition

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
	"github.com/vpptu/ruleflow/pkg/ruleflow/extensions/types"
)

// ─────────────────────────────────────────────
//  PowerFactorCheckCondition — 功率因数检测条件
// ─────────────────────────────────────────────

// PowerFactorCheckCondition 功率因数检测条件
type PowerFactorCheckCondition struct {
	IDValue            string   `json:"id"`
	MinPowerFactor     *float64 `json:"min_power_factor"`     // 功率因数下限
	MaxPowerFactor     *float64 `json:"max_power_factor"`     // 功率因数上限
	ActivePowerPoint   string   `json:"active_power_point"`   // 有功功率数据点名
	ReactivePowerPoint string   `json:"reactive_power_point"` // 无功功率数据点名
	EvaluationMode     string   `json:"evaluation_mode"`      // per_point / computed

	minPF float64
	maxPF float64
}

// NewPowerFactorCheckCondition 创建功率因数检测条件
func NewPowerFactorCheckCondition(id string, minPF, maxPF *float64, activePoint, reactivePoint, mode string) *PowerFactorCheckCondition {
	c := &PowerFactorCheckCondition{
		IDValue:            id,
		ActivePowerPoint:   activePoint,
		ReactivePowerPoint: reactivePoint,
		EvaluationMode:     mode,
	}
	if minPF != nil {
		c.minPF = *minPF
	}
	if maxPF != nil {
		c.maxPF = *maxPF
	}
	if c.EvaluationMode == "" {
		c.EvaluationMode = "computed"
	}
	return c
}

func (c *PowerFactorCheckCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	var pf float64

	switch c.EvaluationMode {
	case "computed":
		mdc, ok := data.(types.MultiDataContextInterface)
		if !ok {
			return false
		}
		p, errP := mdc.GetPoint(c.ActivePowerPoint)
		q, errQ := mdc.GetPoint(c.ReactivePowerPoint)
		if errP != nil || errQ != nil {
			return false
		}
		denom := math.Sqrt(p*p + q*q)
		if denom == 0 {
			return false
		}
		pf = math.Abs(p) / denom

	case "per_point":
		fallthrough
	default:
		pf = data.Value()
	}

	if c.minPF > 0 && pf < c.minPF {
		return true
	}
	if c.maxPF > 0 && pf > c.maxPF {
		return true
	}
	return false
}

func (c *PowerFactorCheckCondition) ID() string   { return c.IDValue }
func (c *PowerFactorCheckCondition) Type() string { return "power_factor_check" }
func (c *PowerFactorCheckCondition) Description() string {
	return fmt.Sprintf("power factor check [%.2f, %.2f]", c.minPF, c.maxPF)
}

// ─────────────────────────────────────────────
//  FrequencyWobbleCondition — 频率波动检测条件
// ─────────────────────────────────────────────

// freqSample 频率采样点
type freqSample struct {
	val float64
	ts  time.Time
}

// FrequencyWobbleCondition 频率波动检测条件
type FrequencyWobbleCondition struct {
	IDValue     string  `json:"id"`
	NominalFreq float64 `json:"nominal_freq"` // 额定频率（Hz）
	Deadband    float64 `json:"deadband"`     // 死区范围（Hz）
	RateLimit   float64 `json:"rate_limit"`   // 频率变化率限制（Hz/s）
	WindowSec   int     `json:"window_sec"`   // 变化率检测窗口（秒）
}

// NewFrequencyWobbleCondition 创建频率波动检测条件
func NewFrequencyWobbleCondition(id string, nominalFreq, deadband, rateLimit float64, windowSec int) *FrequencyWobbleCondition {
	if nominalFreq == 0 {
		nominalFreq = 50.0
	}
	if deadband == 0 {
		deadband = 0.5
	}
	if windowSec == 0 {
		windowSec = 60
	}
	return &FrequencyWobbleCondition{
		IDValue:     id,
		NominalFreq: nominalFreq,
		Deadband:    deadband,
		RateLimit:   rateLimit,
		WindowSec:   windowSec,
	}
}

func (c *FrequencyWobbleCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	freq := data.Value()

	// 1. 稳态越限检测
	lower := c.NominalFreq - c.Deadband
	upper := c.NominalFreq + c.Deadband
	if freq < lower || freq > upper {
		return true
	}

	// 2. 暂态变化率检测
	if c.RateLimit <= 0 || c.WindowSec <= 0 {
		return false
	}

	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false
	}

	key := "freq:" + c.IDValue + ":" + data.DeviceID() + "/" + data.PointName()
	now := time.UnixMilli(data.Timestamp())

	var samples []freqSample
	if raw, loaded := sd.StateStore().Get(key); loaded {
		if s, ok := raw.([]freqSample); ok {
			samples = s
		}
	}

	samples = append(samples, freqSample{val: freq, ts: now})
	cutoff := now.Add(-time.Duration(c.WindowSec) * time.Second)
	i := 0
	for i < len(samples) && samples[i].ts.Before(cutoff) {
		i++
	}
	samples = samples[i:]
	sd.StateStore().Set(key, samples)

	if len(samples) < 2 {
		return false
	}

	first, last := samples[0], samples[len(samples)-1]
	dt := last.ts.Sub(first.ts).Seconds()
	if dt <= 0 {
		return false
	}
	rate := math.Abs(last.val-first.val) / dt
	return rate > c.RateLimit
}

func (c *FrequencyWobbleCondition) ID() string   { return c.IDValue }
func (c *FrequencyWobbleCondition) Type() string { return "frequency_wobble" }
func (c *FrequencyWobbleCondition) Description() string {
	return fmt.Sprintf("freq wobble ±%.1fHz rate<%.2fHz/s", c.Deadband, c.RateLimit)
}

// ─────────────────────────────────────────────
//  RampRateLimitCondition — 爬坡率限制检测条件
// ─────────────────────────────────────────────

// pwSample 功率采样点
type pwSample struct {
	val float64
	ts  time.Time
}

// RampRateLimitCondition 爬坡率限制检测条件
type RampRateLimitCondition struct {
	IDValue     string  `json:"id"`
	MaxRampRate float64 `json:"max_ramp_rate"` // MW/min
	Direction   string  `json:"direction"`     // up / down / both
	WindowSec   int     `json:"window_sec"`    // 滑动窗口（秒）
	RatedPower  float64 `json:"rated_power"`   // 额定功率（MW）
}

// NewRampRateLimitCondition 创建爬坡率限制检测条件
func NewRampRateLimitCondition(id string, maxRampRate float64, direction string, windowSec int, ratedPower float64) *RampRateLimitCondition {
	if direction == "" {
		direction = "both"
	}
	if windowSec == 0 {
		windowSec = 60
	}
	return &RampRateLimitCondition{
		IDValue:     id,
		MaxRampRate: maxRampRate,
		Direction:   direction,
		WindowSec:   windowSec,
		RatedPower:  ratedPower,
	}
}

func (c *RampRateLimitCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false
	}

	key := "ramp:" + c.IDValue + ":" + data.DeviceID() + "/" + data.PointName()
	now := time.UnixMilli(data.Timestamp())
	power := data.Value()

	var samples []pwSample
	if raw, loaded := sd.StateStore().Get(key); loaded {
		if s, ok := raw.([]pwSample); ok {
			samples = s
		}
	}

	samples = append(samples, pwSample{val: power, ts: now})
	cutoff := now.Add(-time.Duration(c.WindowSec) * time.Second)
	i := 0
	for i < len(samples) && samples[i].ts.Before(cutoff) {
		i++
	}
	samples = samples[i:]
	sd.StateStore().Set(key, samples)

	if len(samples) < 2 {
		return false
	}

	first, last := samples[0], samples[len(samples)-1]
	dt := last.ts.Sub(first.ts).Minutes()
	if dt <= 0 {
		return false
	}
	rampRate := (last.val - first.val) / dt

	switch c.Direction {
	case "up":
		if rampRate <= 0 {
			return false
		}
		return rampRate > c.MaxRampRate
	case "down":
		if rampRate >= 0 {
			return false
		}
		return math.Abs(rampRate) > c.MaxRampRate
	case "both":
		fallthrough
	default:
		return math.Abs(rampRate) > c.MaxRampRate
	}
}

func (c *RampRateLimitCondition) ID() string   { return c.IDValue }
func (c *RampRateLimitCondition) Type() string { return "ramp_rate_limit" }
func (c *RampRateLimitCondition) Description() string {
	return fmt.Sprintf("ramp rate limit %.1f MW/min %s", c.MaxRampRate, c.Direction)
}

// ─────────────────────────────────────────────
//  ReversePowerCondition — 反向功率检测条件
// ─────────────────────────────────────────────

// ReversePowerCondition 反向功率检测条件
type ReversePowerCondition struct {
	IDValue     string  `json:"id"`
	Threshold   float64 `json:"threshold"`    // 反向功率阈值（kW）
	PowerPoint  string  `json:"power_point"`  // 有功功率数据点名
	DurationSec int     `json:"duration_sec"` // 持续时间要求（秒）
}

// NewReversePowerCondition 创建反向功率检测条件
func NewReversePowerCondition(id string, threshold float64, powerPoint string, durationSec int) *ReversePowerCondition {
	return &ReversePowerCondition{
		IDValue:     id,
		Threshold:   threshold,
		PowerPoint:  powerPoint,
		DurationSec: durationSec,
	}
}

func (c *ReversePowerCondition) Evaluate(ctx context.Context, data core.DataContext) bool {
	var power float64

	if c.PowerPoint != "" {
		mdc, ok := data.(types.MultiDataContextInterface)
		if !ok {
			return false
		}
		p, err := mdc.GetPoint(c.PowerPoint)
		if err != nil {
			return false
		}
		power = p
	} else {
		power = data.Value()
	}

	isReverse := power < 0 && math.Abs(power) > c.Threshold

	if !isReverse || c.DurationSec <= 0 {
		return isReverse
	}

	sd, ok := data.(core.StatefulDataContext)
	if !ok {
		return false
	}

	key := "revpwr:" + c.IDValue + ":" + data.DeviceID()
	now := time.UnixMilli(data.Timestamp())

	stateI, loaded := sd.StateStore().Get(key)
	if !loaded {
		sd.StateStore().Set(key, &now)
		return false
	}
	since, ok := stateI.(*time.Time)
	if !ok {
		sd.StateStore().Set(key, &now)
		return false
	}
	return now.Sub(*since) >= time.Duration(c.DurationSec)*time.Second
}

func (c *ReversePowerCondition) ID() string   { return c.IDValue }
func (c *ReversePowerCondition) Type() string { return "reverse_power" }
func (c *ReversePowerCondition) Description() string {
	return fmt.Sprintf("reverse power >%.1fkW dur=%ds", c.Threshold, c.DurationSec)
}