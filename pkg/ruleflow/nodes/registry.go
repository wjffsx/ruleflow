// Package nodes provides the node registry with on-demand registration.
//
// V7 Refactoring:
//   - Moved from registry/registry.go to nodes/registry.go
//   - Fixed dependency inversion: registry no longer imports builtin/extensions
//   - Added on-demand registration: NewEmptyRegistry() + RegisterBuiltin() + RegisterVPP()
//   - Component metadata moved to nodes/meta.go
//
// Dependency direction (correct):
//   - builtin → nodes/registry → core
//   - extensions → nodes/registry → core
package nodes

import (
	"fmt"
	"sync"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ConditionFactory condition factory function
type ConditionFactory func(id string, config map[string]any) (core.Condition, error)

// ActionFactory action factory function
type ActionFactory func(id string, config map[string]any) (core.Action, error)

// NodePackage node package interface for on-demand registration
type NodePackage interface {
	// GetConditionFactories returns condition factories
	GetConditionFactories() map[string]ConditionFactory
	// GetActionFactories returns action factories
	GetActionFactories() map[string]ActionFactory
}

// Registry condition/action registry (instance-level, not global)
type Registry struct {
	conditions map[string]ConditionFactory
	actions    map[string]ActionFactory
	mu         sync.RWMutex
}

// NewEmptyRegistry creates an empty registry (zero nodes).
// Use this for on-demand registration pattern.
func NewEmptyRegistry() *Registry {
	return &Registry{
		conditions: make(map[string]ConditionFactory),
		actions:    make(map[string]ActionFactory),
	}
}

// RegisterCondition registers a condition factory
func (r *Registry) RegisterCondition(typeName string, factory ConditionFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conditions[typeName] = factory
}

// RegisterAction registers an action factory
func (r *Registry) RegisterAction(typeName string, factory ActionFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions[typeName] = factory
}

// RegisterPackage registers all nodes from a NodePackage.
// Use this for on-demand registration of builtin or VPP nodes.
func (r *Registry) RegisterPackage(pkg NodePackage) {
	for typeName, factory := range pkg.GetConditionFactories() {
		r.RegisterCondition(typeName, factory)
	}
	for typeName, factory := range pkg.GetActionFactories() {
		r.RegisterAction(typeName, factory)
	}
}

// RegisterConditions registers multiple condition factories
func (r *Registry) RegisterConditions(factories map[string]ConditionFactory) {
	for typeName, factory := range factories {
		r.RegisterCondition(typeName, factory)
	}
}

// RegisterActions registers multiple action factories
func (r *Registry) RegisterActions(factories map[string]ActionFactory) {
	for typeName, factory := range factories {
		r.RegisterAction(typeName, factory)
	}
}

// CreateCondition creates a condition instance
func (r *Registry) CreateCondition(typeName, id string, config map[string]any) (core.Condition, error) {
	r.mu.RLock()
	factory, ok := r.conditions[typeName]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown condition type: %s", typeName)
	}
	return factory(id, config)
}

// CreateAction creates an action instance
func (r *Registry) CreateAction(typeName, id string, config map[string]any) (core.Action, error) {
	r.mu.RLock()
	factory, ok := r.actions[typeName]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown action type: %s", typeName)
	}
	return factory(id, config)
}

// ListConditionTypes lists registered condition types
func (r *Registry) ListConditionTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.conditions))
	for k := range r.conditions {
		types = append(types, k)
	}
	return types
}

// ListActionTypes lists registered action types
func (r *Registry) ListActionTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.actions))
	for k := range r.actions {
		types = append(types, k)
	}
	return types
}

// HasCondition checks if a condition type is registered
func (r *Registry) HasCondition(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conditions[typeName] != nil
}

// HasAction checks if an action type is registered
func (r *Registry) HasAction(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.actions[typeName] != nil
}