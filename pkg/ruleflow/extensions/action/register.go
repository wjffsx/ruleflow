// Package action provides VPP action nodes registration
//
// V7 Refactoring: Updated to use nodes/util instead of extensions/internal/util.
package action

import (
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes/util"
)

// GetFactories 返回所有 VPP 动作节点的工厂函数
func GetFactories() map[string]func(id string, config map[string]any) (core.Action, error) {
	return map[string]func(id string, config map[string]any) (core.Action, error){
		"aggregator":         newAggregatorAction,
		"dispatch_control":   newDispatchControlAction,
		"market_price_query": newMarketPriceQueryAction,
		"carbon_calc":        newCarbonCalcAction,
		"weather_query":      newWeatherQueryAction,
		"delta_accumulator":  newDeltaAccumulatorAction,
		"efficiency_calc":    newEfficiencyCalcAction,
		"plant_split":        newPlantSplitAction,
	}
}

// ─────────────────────────────────────────────
//  工厂函数
// ─────────────────────────────────────────────

func newAggregatorAction(id string, config map[string]any) (core.Action, error) {
	groupKey, _ := config["group_key"].(string)
	method, _ := config["method"].(string)
	var inputPoints []string
	if raw, ok := config["input_points"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				inputPoints = append(inputPoints, s)
			}
		}
	}
	outputPoint, _ := config["output_point"].(string)
	windowSec := 0
	if v, ok := config["window_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			windowSec = int(f)
		}
	}
	return NewAggregatorAction(id, groupKey, method, inputPoints, outputPoint, windowSec), nil
}

func newDispatchControlAction(id string, config map[string]any) (core.Action, error) {
	target, _ := config["target"].(string)
	command, _ := config["command"].(string)
	param := 0.0
	if v, ok := config["param"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			param = f
		}
	}
	protocol, _ := config["protocol"].(string)
	timeoutMs := 5000
	if v, ok := config["timeout_ms"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			timeoutMs = int(f)
		}
	}
	retryCount := 0
	if v, ok := config["retry_count"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			retryCount = int(f)
		}
	}
	return NewDispatchControlAction(id, target, command, param, protocol, timeoutMs, retryCount, nil), nil
}

func newMarketPriceQueryAction(id string, config map[string]any) (core.Action, error) {
	market, _ := config["market"].(string)
	product, _ := config["product"].(string)
	region, _ := config["region"].(string)
	cacheTTLSec := 300
	if v, ok := config["cache_ttl_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			cacheTTLSec = int(f)
		}
	}
	outputTag, _ := config["output_tag"].(string)
	endpoint, _ := config["endpoint"].(string)
	return NewMarketPriceQueryAction(id, market, product, region, cacheTTLSec, outputTag, endpoint, nil), nil
}

func newCarbonCalcAction(id string, config map[string]any) (core.Action, error) {
	scope, _ := config["scope"].(string)
	emissionFactor := 0.0
	if v, ok := config["emission_factor"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			emissionFactor = f
		}
	}
	fuelType, _ := config["fuel_type"].(string)
	powerPoint, _ := config["power_point"].(string)
	outputPoint, _ := config["output_point"].(string)
	return NewCarbonCalcAction(id, scope, emissionFactor, fuelType, powerPoint, outputPoint), nil
}

func newWeatherQueryAction(id string, config map[string]any) (core.Action, error) {
	weatherAPI, _ := config["weather_api"].(string)
	var lat, lon *float64
	if v, ok := config["latitude"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			lat = &f
		}
	}
	if v, ok := config["longitude"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			lon = &f
		}
	}
	var fields []string
	if raw, ok := config["fields"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				fields = append(fields, s)
			}
		}
	}
	outputPrefix, _ := config["output_prefix"].(string)
	cacheTTLSec := 600
	if v, ok := config["cache_ttl_sec"]; ok {
		if f, ok := util.ToFloat64(v); ok {
			cacheTTLSec = int(f)
		}
	}
	return NewWeatherQueryAction(id, weatherAPI, lat, lon, fields, outputPrefix, cacheTTLSec, nil), nil
}

func newDeltaAccumulatorAction(id string, config map[string]any) (core.Action, error) {
	key, _ := config["key"].(string)
	period, _ := config["period"].(string)
	sourcePoint, _ := config["source_point"].(string)
	outputPoint, _ := config["output_point"].(string)
	resetAtMidnight := true
	if v, ok := config["reset_at_midnight"]; ok {
		if b, ok := v.(bool); ok {
			resetAtMidnight = b
		}
	}
	return NewDeltaAccumulatorAction(id, key, period, sourcePoint, outputPoint, resetAtMidnight), nil
}

func newEfficiencyCalcAction(id string, config map[string]any) (core.Action, error) {
	inputPoint, _ := config["input_point"].(string)
	outputPoint, _ := config["output_point"].(string)
	efficiencyPoint, _ := config["efficiency_point"].(string)
	outputUnit, _ := config["output_unit"].(string)
	return NewEfficiencyCalcAction(id, inputPoint, outputPoint, efficiencyPoint, outputUnit), nil
}

func newPlantSplitAction(id string, config map[string]any) (core.Action, error) {
	splitBy, _ := config["split_by"].(string)
	keySource, _ := config["key_source"].(string)
	var outputRoutes []string
	if raw, ok := config["output_routes"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				outputRoutes = append(outputRoutes, s)
			}
		}
	}
	return NewPlantSplitAction(id, splitBy, keySource, outputRoutes), nil
}
