// Package builtin provides IoT-generic rule nodes.
package builtin

import (
	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/action"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/builtin/condition"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
)

// Package builtin node package for registry registration
type Package struct{}

// 编译期接口检查：builtin.Package 实现 core.NodePackage
var _ core.NodePackage = Package{}

// GetConditionFactories returns all builtin condition factories
func (Package) GetConditionFactories() map[string]core.ConditionFactory {
	result := make(map[string]core.ConditionFactory)
	for k, v := range condition.GetFactories() {
		result[k] = v
	}
	return result
}

// GetActionFactories returns all builtin action factories
func (Package) GetActionFactories() map[string]core.ActionFactory {
	result := make(map[string]core.ActionFactory)
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
