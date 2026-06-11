// Package extensions provides VPP (Virtual Power Plant) business extension interfaces.
//
// V7 Refactoring: Moved from core/vpp/interfaces.go.
// These interfaces are VPP business-specific, not core engine abstractions.
// Placing them under extensions maintains correct layer boundaries.
//
// Note: For use within action/condition/flow sub-packages, import extensions/types
// to avoid import cycles.
package extensions

import (
	"github.com/vpptu/ruleflow/pkg/ruleflow/extensions/types"
)

// MultiDataContextInterface multi-data-point context interface (alias for types.MultiDataContextInterface)
type MultiDataContextInterface = types.MultiDataContextInterface

// SubChainEngine sub-rule-chain execution engine interface (alias for types.SubChainEngine)
type SubChainEngine = types.SubChainEngine

// CommandDispatcher command dispatch interface (alias for types.CommandDispatcher)
type CommandDispatcher = types.CommandDispatcher
