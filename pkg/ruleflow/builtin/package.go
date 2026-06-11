// Package builtin provides IoT-generic rule nodes.
package builtin

import (
	"github.com/vpptu/ruleflow/pkg/ruleflow/builtin/action"
	"github.com/vpptu/ruleflow/pkg/ruleflow/builtin/condition"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes"
)

// Package builtin node package for registry registration
type Package struct{}

// GetConditionFactories returns all builtin condition factories
func (Package) GetConditionFactories() map[string]nodes.ConditionFactory {
	result := make(map[string]nodes.ConditionFactory)
	for k, v := range condition.GetFactories() {
		result[k] = v
	}
	return result
}

// GetActionFactories returns all builtin action factories
func (Package) GetActionFactories() map[string]nodes.ActionFactory {
	result := make(map[string]nodes.ActionFactory)
	for k, v := range action.GetFactories() {
		result[k] = v
	}
	return result
}

// Builtin is the singleton Package instance
var Builtin = Package{}

// RegisterAll registers all builtin nodes to a registry.
// Convenience function for one-line registration.
func RegisterAll(r *nodes.Registry) {
	r.RegisterPackage(Builtin)
}