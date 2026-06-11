// Package types provides VPP business extension interfaces.
//
// V7 Refactoring: Moved from extensions/interfaces.go to avoid import cycles.
// Sub-packages (action/condition/flow) import this types package,
// while extensions/register.go imports sub-packages.
package types

import (
	"context"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// MultiDataContextInterface multi-data-point context interface.
// Used for rules requiring multi-input aggregation (e.g., power factor calculation, demand response detection).
type MultiDataContextInterface interface {
	core.DataContext
	// GetPoint gets the value of an associated data point
	GetPoint(pointName string) (float64, error)
	// GetPointData gets the complete DataContext of an associated data point
	GetPointData(pointName string) (core.DataContext, bool)
	// GetAllPoints gets all data point names
	GetAllPoints() []string
	// SetPointValue sets the value of an associated data point
	SetPointValue(pointName string, value float64)
}

// SubChainEngine sub-rule-chain execution engine interface
type SubChainEngine interface {
	// ExecuteChain executes a rule chain by ID
	ExecuteChain(ctx context.Context, chainID string, data core.DataContext) error
}

// CommandDispatcher command dispatch interface (adapter pattern)
type CommandDispatcher interface {
	Dispatch(ctx context.Context, target, command string, param float64, protocol string, timeout time.Duration) error
}