// Package condition provides VPP condition nodes registration
//
// V7 Refactoring: Updated to use nodes/util instead of extensions/internal/util.
package condition

import (
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes/util"
	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// GetFactories 返回所有 VPP 条件节点的工厂函数
func GetFactories() map[string]func(id string, config map[string]any) (core.Condition, error) {
	return map[string]func(id string, config map[string]any) (core.Condition, error){
		"soc_monitor":           newSOCMonitorCondition,
		"power_factor_check":    newPowerFactorCheckCondition,
		"frequency_wobble":      newFrequencyWobbleCondition,
		"ramp_rate_limit":       newRampRateLimitCondition,
		"reverse_power":         newReversePowerCondition,
		"time_of_use_price":     newToUPriceCondition,
		"demand_response_check": newDemandResponseCheckCondition,
		"soh_estimator":         newSOHEstimatorCondition,
	}
}

// ─────────────────────────────────────────────
//  工厂函数
// ─────────────────────────────────────────────

func newSOCMonitorCondition(id string, config map[string]any) (core.Condition, error) {
	var chargeHigh, chargeLow, dischargeHigh, dischargeLow *float64
	if v, ok := config["charge_high_limit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			chargeHigh = &f
		}
	}
	if v, ok := config["charge_low_limit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			chargeLow = &f
		}
	}
	if v, ok := config["discharge_high_limit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			dischargeHigh = &f
		}
	}
	if v, ok := config["discharge_low_limit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			dischargeLow = &f
		}
	}
	chargeStatePoint, _ := config["charge_state_point"].(string)
	return NewSOCMonitorCondition(id, chargeHigh, chargeLow, dischargeHigh, dischargeLow, chargeStatePoint), nil
}

func newPowerFactorCheckCondition(id string, config map[string]any) (core.Condition, error) {
	var minPF, maxPF *float64
	if v, ok := config["min_power_factor"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			minPF = &f
		}
	}
	if v, ok := config["max_power_factor"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			maxPF = &f
		}
	}
	activePoint, _ := config["active_power_point"].(string)
	reactivePoint, _ := config["reactive_power_point"].(string)
	mode, _ := config["evaluation_mode"].(string)
	return NewPowerFactorCheckCondition(id, minPF, maxPF, activePoint, reactivePoint, mode), nil
}

func newFrequencyWobbleCondition(id string, config map[string]any) (core.Condition, error) {
	nominalFreq := 50.0
	if v, ok := config["nominal_freq"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			nominalFreq = f
		}
	}
	deadband := 0.5
	if v, ok := config["deadband"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			deadband = f
		}
	}
	rateLimit := 0.0
	if v, ok := config["rate_limit"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			rateLimit = f
		}
	}
	windowSec := 60
	if v, ok := config["window_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			windowSec = int(f)
		}
	}
	return NewFrequencyWobbleCondition(id, nominalFreq, deadband, rateLimit, windowSec), nil
}

func newRampRateLimitCondition(id string, config map[string]any) (core.Condition, error) {
	maxRampRate := 0.0
	if v, ok := config["max_ramp_rate"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			maxRampRate = f
		}
	}
	direction, _ := config["direction"].(string)
	windowSec := 60
	if v, ok := config["window_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			windowSec = int(f)
		}
	}
	ratedPower := 0.0
	if v, ok := config["rated_power"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			ratedPower = f
		}
	}
	return NewRampRateLimitCondition(id, maxRampRate, direction, windowSec, ratedPower), nil
}

func newReversePowerCondition(id string, config map[string]any) (core.Condition, error) {
	threshold := 0.0
	if v, ok := config["threshold"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			threshold = f
		}
	}
	powerPoint, _ := config["power_point"].(string)
	durationSec := 0
	if v, ok := config["duration_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			durationSec = int(f)
		}
	}
	return NewReversePowerCondition(id, threshold, powerPoint, durationSec), nil
}

func newToUPriceCondition(id string, config map[string]any) (core.Condition, error) {
	period, _ := config["period"].(string)
	timezone, _ := config["timezone"].(string)
	season, _ := config["season"].(string)
	region, _ := config["region"].(string)
	return NewToUPriceCondition(id, period, timezone, season, region), nil
}

func newDemandResponseCheckCondition(id string, config map[string]any) (core.Condition, error) {
	expectedKW := 0.0
	if v, ok := config["expected_reduction_kw"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			expectedKW = f
		}
	}
	baselinePoint, _ := config["baseline_point"].(string)
	actualPoint, _ := config["actual_point"].(string)
	tolerancePct := 10.0
	if v, ok := config["tolerance_pct"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			tolerancePct = f
		}
	}
	return NewDemandResponseCheckCondition(id, expectedKW, baselinePoint, actualPoint, tolerancePct), nil
}

func newSOHEstimatorCondition(id string, config map[string]any) (core.Condition, error) {
	sohThreshold := 80.0
	if v, ok := config["soh_threshold"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			sohThreshold = f
		}
	}
	cycleCountPoint, _ := config["cycle_count_point"].(string)
	irPoint, _ := config["internal_resistance_point"].(string)
	capacityPoint, _ := config["capacity_point"].(string)
	nominalCapacity := 0.0
	if v, ok := config["nominal_capacity"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			nominalCapacity = f
		}
	}
	method, _ := config["method"].(string)
	return NewSOHEstimatorCondition(id, sohThreshold, cycleCountPoint, irPoint, capacityPoint, nominalCapacity, method), nil
}