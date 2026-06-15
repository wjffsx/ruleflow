// Package util provides shared utility functions for node implementations.
//
// V7 Refactoring: Merged from builtin/internal/util and extensions/internal/util.
// This package is intentionally placed under nodes/ (not internal/) to allow
// cross-package usage between builtin and extensions while maintaining
// proper dependency direction.
//
// V14 Optimization: Added StateKey function for unified state key format.
package util

import "fmt"

// ToFloat64 converts any numeric type to float64.
// Supports float64/float32/int/int64/int32, returns false for other types.
func ToFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// ─────────────────────────────────────────────
//  State Key 统一格式（V14）
// ─────────────────────────────────────────────

// StateKey generates a unified state key for stateful nodes.
//
// V14 Architecture:
//   - Unified format: {node_type}:{node_id}:{device_id}/{point_name}
//   - Ensures consistent state management across all stateful nodes
//   - Allows distinguishing multiple condition instances for same data point
//
// Example:
//   - StateKey("duration", "cond_001", "device001", "power")
//     → "duration:cond_001:device001/power"
//   - StateKey("trend", "cond_002", "device001", "frequency")
//     → "trend:cond_002:device001/frequency"
func StateKey(nodeType, nodeID, deviceID, pointName string) string {
	return fmt.Sprintf("%s:%s:%s/%s", nodeType, nodeID, deviceID, pointName)
}

// StateKeySimple generates a simplified state key without nodeID.
// Used for nodes that don't need to distinguish multiple instances.
//
// Format: {node_type}:{device_id}/{point_name}
func StateKeySimple(nodeType, deviceID, pointName string) string {
	return fmt.Sprintf("%s:%s/%s", nodeType, deviceID, pointName)
}
