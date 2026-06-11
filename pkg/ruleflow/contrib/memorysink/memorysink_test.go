package memorysink

import (
	"testing"
	"time"
)

// ─────────────────────────────────────────────
//  MemorySink 测试
// ─────────────────────────────────────────────

func TestMemorySink_RecordsAllEvents(t *testing.T) {
	s := NewMemorySink()

	s.IncEvalTotal("c1", "matched")
	s.IncEvalTotal("c1", "matched")
	s.IncEvalTotal("c1", "unmatched")
	s.IncActionTotal("c1", "rule", "ok")
	s.IncActionTotal("c1", "rule", "errored")
	s.IncRuleThrottled("c1", "r1")
	s.IncPanic("c1", "r2")
	s.ObserveEvalDuration("c1", 100*time.Millisecond)
	s.ObserveEvalDuration("c1", 200*time.Millisecond)
	s.SetLoadedChains(7)
	s.SetActiveEval(4)

	if got := s.EvalTotal["c1"]["matched"]; got != 2 {
		t.Errorf("matched count: want 2, got %d", got)
	}
	if got := s.EvalTotal["c1"]["unmatched"]; got != 1 {
		t.Errorf("unmatched count: want 1, got %d", got)
	}
	if got := s.ActionTotal["c1"]["rule"]["ok"]; got != 1 {
		t.Errorf("action ok: want 1, got %d", got)
	}
	if got := s.ActionTotal["c1"]["rule"]["errored"]; got != 1 {
		t.Errorf("action errored: want 1, got %d", got)
	}
	if got := s.RuleThrottled["c1"]["r1"]; got != 1 {
		t.Errorf("throttled: want 1, got %d", got)
	}
	if got := s.Panic["c1"]["r2"]; got != 1 {
		t.Errorf("panic: want 1, got %d", got)
	}
	if got := s.EvalDuration["c1"]; got != 300*time.Millisecond.Nanoseconds() {
		t.Errorf("eval duration: want 300ms, got %v", time.Duration(got))
	}
	if got := s.EvalCount["c1"]; got != 2 {
		t.Errorf("eval count: want 2, got %d", got)
	}
	if s.LoadedChains != 7 {
		t.Errorf("loaded chains: want 7, got %d", s.LoadedChains)
	}
	if s.ActiveEval != 4 {
		t.Errorf("active eval: want 4, got %d", s.ActiveEval)
	}
}

func TestMemorySink_SnapshotIsConsistent(t *testing.T) {
	s := NewMemorySink()
	s.IncEvalTotal("c1", "matched")
	s.SetLoadedChains(3)

	snap := s.Snapshot()
	if snap.EvalTotal["c1"]["matched"] != 1 {
		t.Errorf("snapshot matched: want 1, got %d", snap.EvalTotal["c1"]["matched"])
	}
	if snap.LoadedChains != 3 {
		t.Errorf("snapshot loaded chains: want 3, got %d", snap.LoadedChains)
	}

	// 修改 sink 不应影响已拿到的 snapshot
	s.IncEvalTotal("c1", "matched")
	s.SetLoadedChains(10)
	if snap.EvalTotal["c1"]["matched"] != 1 {
		t.Error("snapshot should be independent of sink mutations")
	}
	if snap.LoadedChains != 3 {
		t.Error("snapshot should be independent of sink mutations")
	}
}

func TestMemorySink_ObserveConditionAndAction(t *testing.T) {
	s := NewMemorySink()

	s.ObserveConditionEval("c1", "n1", 50*time.Millisecond, true)
	s.ObserveConditionEval("c1", "n1", 30*time.Millisecond, false)
	s.ObserveActionExec("c1", "n2", "http", 100*time.Millisecond, false)
	s.ObserveActionExec("c1", "n2", "http", 200*time.Millisecond, true)

	snap := s.Snapshot()
	if snap.ConditionEval["c1"]["n1"]["count"] != 2 {
		t.Errorf("condition count: want 2, got %d", snap.ConditionEval["c1"]["n1"]["count"])
	}
	if snap.ConditionEval["c1"]["n1"]["matched"] != 1 {
		t.Errorf("condition matched: want 1, got %d", snap.ConditionEval["c1"]["n1"]["matched"])
	}
	if snap.ActionExec["c1"]["n2"]["count"] != 2 {
		t.Errorf("action count: want 2, got %d", snap.ActionExec["c1"]["n2"]["count"])
	}
	if snap.ActionExec["c1"]["n2"]["error"] != 1 {
		t.Errorf("action error: want 1, got %d", snap.ActionExec["c1"]["n2"]["error"])
	}
}
