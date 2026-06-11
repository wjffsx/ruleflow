// Package builtin provides builtin condition and action nodes for ruleflow.
//
// This package has been refactored into sub-packages:
//   - builtin/condition: condition nodes (device, point, value, quality, limit, time)
//   - builtin/condition/stateful: stateful condition nodes (state_change, duration, trend, periodic, dynamic_threshold)
//   - builtin/action: action nodes (transform, rename, tag, drop, route, limit_check, quality_mark, alarm, delay)
//   - extensions: VPP-specific nodes (Phase 1)
//
// Use the sub-packages directly:
//
//	import "github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
//	import "github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/action"
package builtin
