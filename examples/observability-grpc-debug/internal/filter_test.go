package debuginternal

import (
	"testing"

	coredebug "github.com/vpptu/ruleflow/pkg/ruleflow/debug"
)

func makeEvent(chainID, nodeID, relationType string) coredebug.DebugEvent {
	return coredebug.DebugEvent{
		ChainID:      chainID,
		NodeID:       nodeID,
		RelationType: relationType,
	}
}

func TestSubscribeFilter_EmptyMatchesAll(t *testing.T) {
	f := NewSubscribeFilter(nil, nil, "", false)
	if !f.Match(makeEvent("chain-1", "node-1", "matched")) {
		t.Error("empty filter should match all")
	}
}

func TestSubscribeFilter_FilterChainID(t *testing.T) {
	f := NewSubscribeFilter([]string{"chain-1"}, nil, "", false)
	if !f.Match(makeEvent("chain-1", "n1", "matched")) {
		t.Error("should match chain-1")
	}
	if f.Match(makeEvent("chain-2", "n1", "matched")) {
		t.Error("should NOT match chain-2")
	}
}

func TestSubscribeFilter_FilterRuleIDs(t *testing.T) {
	f := NewSubscribeFilter(nil, []string{"rule-1", "rule-2"}, "", false)
	tests := []struct {
		ruleID string
		match  bool
	}{
		{"rule-1", true},
		{"rule-2", true},
		{"rule-3", false},
	}
	for _, tc := range tests {
		ev := makeEvent("c1", "n1", "matched")
		ev.RuleID = tc.ruleID
		if got := f.Match(ev); got != tc.match {
			t.Errorf("ruleID=%s expect match=%v got=%v", tc.ruleID, tc.match, got)
		}
	}
}

func TestSubscribeFilter_FilterNodeType(t *testing.T) {
	f := NewSubscribeFilter(nil, nil, "condition", false)
	if !f.Match(withNodeType(makeEvent("c1", "n1", "matched"), "condition")) {
		t.Error("should match condition node")
	}
	if f.Match(withNodeType(makeEvent("c1", "n1", "matched"), "action:transform")) {
		t.Error("should NOT match action node")
	}
}

func TestSubscribeFilter_OnlyErrors(t *testing.T) {
	f := NewSubscribeFilter(nil, nil, "", true)
	if !f.Match(makeEvent("c1", "n1", "error")) {
		t.Error("should match error relation")
	}
	if !f.Match(makeEvent("c1", "n1", "dropped")) {
		t.Error("should match dropped relation")
	}
	if f.Match(makeEvent("c1", "n1", "matched")) {
		t.Error("should NOT match matched")
	}
	if f.Match(makeEvent("c1", "n1", "unmatched")) {
		t.Error("should NOT match unmatched")
	}
}

func TestSubscribeFilter_Combined(t *testing.T) {
	f := NewSubscribeFilter([]string{"chain-1"}, nil, "condition", true)
	// 必须同时满足所有条件
	ev := withNodeType(makeEvent("chain-1", "n1", "error"), "condition")
	if !f.Match(ev) {
		t.Error("should match when all conditions met")
	}
	ev2 := withNodeType(makeEvent("chain-1", "n1", "matched"), "condition")
	if f.Match(ev2) {
		t.Error("should NOT match matched when only_errors=true")
	}
}

// withNodeType 辅助函数
func withNodeType(e coredebug.DebugEvent, nt string) coredebug.DebugEvent {
	e.NodeType = nt
	return e
}
