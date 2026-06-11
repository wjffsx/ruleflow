// Package extensions provides VPP (Virtual Power Plant) specific rule nodes.
package extensions

import (
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/action"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/condition"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/extensions/flow"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
)

// Package VPP node package for registry registration
type Package struct{}

// GetConditionFactories returns all VPP condition factories (NodePackage interface)
func (Package) GetConditionFactories() map[string]nodes.ConditionFactory {
	result := make(map[string]nodes.ConditionFactory)
	for k, v := range condition.GetFactories() {
		result[k] = v
	}
	return result
}

// GetActionFactories returns all VPP action factories (NodePackage interface)
func (Package) GetActionFactories() map[string]nodes.ActionFactory {
	result := make(map[string]nodes.ActionFactory)
	for k, v := range action.GetFactories() {
		result[k] = v
	}
	// Merge flow nodes
	for k, v := range flow.GetFactories() {
		result[k] = v
	}
	return result
}

// VPP is the singleton Package instance
var VPP = Package{}

// RegisterAll registers all VPP nodes to a registry.
// Convenience function for one-line registration.
func RegisterAll(r *nodes.Registry) {
	r.RegisterPackage(VPP)
}