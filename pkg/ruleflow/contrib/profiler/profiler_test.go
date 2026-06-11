package profiler

import (
	"math"
	"testing"
)

func TestProfiler_Record(t *testing.T) {
	p := NewProfiler()

	p.Record("chain-1", "node-1", "condition", 100, false)
	p.Record("chain-1", "node-1", "condition", 200, false)

	p.Record("chain-1", "node-1", "action", 500, false)
	p.Record("chain-1", "node-1", "action", 1000, true)

	snap := p.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(snap))
	}

	for _, prof := range snap {
		if prof.NodeType == "condition" {
			if prof.ExecCount != 2 {
				t.Errorf("condition exec_count: want 2, got %d", prof.ExecCount)
			}
			if prof.MaxLatencyNs != 200 {
				t.Errorf("condition max_latency_ns: want 200, got %d", prof.MaxLatencyNs)
			}
			if math.Abs(prof.AvgLatencyNs-150) > 1 {
				t.Errorf("condition avg_latency_ns: want ~150, got %f", prof.AvgLatencyNs)
			}
			if prof.ErrorCount != 0 {
				t.Errorf("condition error_count: want 0, got %d", prof.ErrorCount)
			}
		}
		if prof.NodeType == "action" {
			if prof.ExecCount != 2 {
				t.Errorf("action exec_count: want 2, got %d", prof.ExecCount)
			}
			if prof.MaxLatencyNs != 1000 {
				t.Errorf("action max_latency_ns: want 1000, got %d", prof.MaxLatencyNs)
			}
			if math.Abs(prof.AvgLatencyNs-750) > 1 {
				t.Errorf("action avg_latency_ns: want ~750, got %f", prof.AvgLatencyNs)
			}
			if prof.ErrorCount != 1 {
				t.Errorf("action error_count: want 1, got %d", prof.ErrorCount)
			}
		}
	}
}

func TestProfiler_TopByMaxLatency(t *testing.T) {
	p := NewProfiler()

	p.Record("c1", "fast", "condition", 10, false)
	p.Record("c1", "medium", "condition", 100, false)
	p.Record("c1", "slow", "condition", 1000, false)

	top := p.TopByMaxLatency(2)
	if len(top) != 2 {
		t.Fatalf("TopByMaxLatency(2): want 2, got %d", len(top))
	}
	if top[0].NodeID != "slow" || top[0].MaxLatencyNs != 1000 {
		t.Errorf("top[0] should be 'slow' with max 1000, got %s(%d)", top[0].NodeID, top[0].MaxLatencyNs)
	}
	if top[1].NodeID != "medium" || top[1].MaxLatencyNs != 100 {
		t.Errorf("top[1] should be 'medium' with max 100, got %s(%d)", top[1].NodeID, top[1].MaxLatencyNs)
	}

	all := p.TopByMaxLatency(0)
	if len(all) != 3 {
		t.Errorf("TopByMaxLatency(0): want 3, got %d", len(all))
	}
}

func TestProfiler_TopByAvgLatency(t *testing.T) {
	p := NewProfiler()

	p.Record("c1", "r1", "condition", 100, false)
	p.Record("c1", "r2", "condition", 300, false)
	p.Record("c1", "r3", "condition", 200, false)

	top := p.TopByAvgLatency(2)
	if len(top) != 2 {
		t.Fatalf("TopByAvgLatency(2): want 2, got %d", len(top))
	}
	if top[0].NodeID != "r2" {
		t.Errorf("top[0] should be 'r2' (highest avg), got %s", top[0].NodeID)
	}
}

func TestProfiler_TopByExecCount(t *testing.T) {
	p := NewProfiler()

	p.Record("c1", "r1", "condition", 10, false)
	p.Record("c1", "r1", "condition", 10, false)
	p.Record("c1", "r1", "condition", 10, false)

	p.Record("c1", "r2", "condition", 10, false)
	p.Record("c1", "r2", "condition", 10, false)

	p.Record("c1", "r3", "condition", 10, false)

	top := p.TopByExecCount(2)
	if len(top) != 2 {
		t.Fatalf("TopByExecCount(2): want 2, got %d", len(top))
	}
	if top[0].NodeID != "r1" || top[0].ExecCount != 3 {
		t.Errorf("top[0] should be 'r1' with exec_count 3, got %s(%d)", top[0].NodeID, top[0].ExecCount)
	}
	if top[1].NodeID != "r2" || top[1].ExecCount != 2 {
		t.Errorf("top[1] should be 'r2' with exec_count 2, got %s(%d)", top[1].NodeID, top[1].ExecCount)
	}
}

func TestProfiler_EmptySnapshot(t *testing.T) {
	p := NewProfiler()

	snap := p.Snapshot()
	if snap == nil {
		t.Fatal("Snapshot() should return non-nil slice")
	}
	if len(snap) != 0 {
		t.Errorf("empty profiler snapshot: want 0, got %d", len(snap))
	}
}

func TestProfiler_Reset(t *testing.T) {
	p := NewProfiler()

	p.Record("c1", "r1", "condition", 100, false)
	p.Record("c1", "r2", "action", 200, false)

	if len(p.Snapshot()) != 2 {
		t.Fatal("expected 2 profiles before reset")
	}

	p.Reset()

	snap := p.Snapshot()
	if len(snap) != 0 {
		t.Errorf("after Reset(): want 0 profiles, got %d", len(snap))
	}

	p.Record("c1", "r1", "condition", 300, true)
	if len(p.Snapshot()) != 1 {
		t.Errorf("after re-record: want 1 profile, got %d", len(p.Snapshot()))
	}
}

func TestProfiler_ConcurrentRecord(t *testing.T) {
	p := NewProfiler()
	done := make(chan struct{})

	const goroutines = 10
	const recordsPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < recordsPerGoroutine; j++ {
				nodeID := "node"
				if j%2 == 0 {
					nodeID = "node-even"
				}
				p.Record("chain-main", nodeID, "condition", int64(j), j%5 == 0)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	snap := p.Snapshot()
	if len(snap) == 0 {
		t.Fatal("expected profiles after concurrent recording")
	}

	for _, prof := range snap {
		if prof.ExecCount <= 0 {
			t.Errorf("profiler %s/%s has exec_count %d", prof.NodeID, prof.NodeType, prof.ExecCount)
		}
	}
}
