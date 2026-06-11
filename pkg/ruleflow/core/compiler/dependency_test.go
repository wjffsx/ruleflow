package compiler

import (
	"errors"
	"sync"
	"testing"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  依赖图测试
// ─────────────────────────────────────────────

func TestDependencyGraph_AddEdge_NoCycle(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")

	if err := g.AddEdge("A", "B"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := g.AddEdge("B", "C"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	deps := g.GetDependencies("A")
	if len(deps) != 1 || deps[0] != "B" {
		t.Errorf("expected A->[B], got %v", deps)
	}
}

func TestDependencyGraph_AddEdge_DirectCycle(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")

	err := g.AddEdge("A", "A")
	if err == nil {
		t.Fatal("expected error for self-cycle")
	}
	if !errors.Is(err, core.ErrCyclicDependency) {
		t.Errorf("expected ErrCyclicDependency, got %v", err)
	}
}

func TestDependencyGraph_AddEdge_IndirectCycle(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")

	g.AddEdge("A", "B")
	g.AddEdge("B", "C")

	// C -> A 会形成 A->B->C->A
	err := g.AddEdge("C", "A")
	if err == nil {
		t.Fatal("expected error for indirect cycle")
	}
	if !errors.Is(err, core.ErrCyclicDependency) {
		t.Errorf("expected ErrCyclicDependency, got %v", err)
	}
}

func TestDependencyGraph_GetReferencedBy(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")

	g.AddEdge("A", "C")
	g.AddEdge("B", "C")

	refs := g.GetReferencedBy("C")
	if len(refs) != 2 {
		t.Errorf("expected 2 references, got %v", refs)
	}
}

func TestDependencyGraph_RemoveNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddEdge("A", "B")

	g.RemoveNode("A")

	if g.HasNode("A") {
		t.Error("A should be removed")
	}
	if !g.HasNode("B") {
		t.Error("B should still exist")
	}
	// A->B 边被清理
	refs := g.GetReferencedBy("B")
	if len(refs) != 0 {
		t.Errorf("expected no references, got %v", refs)
	}
}

func TestDependencyGraph_TopoSort(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")
	g.AddNode("D")

	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("B", "D")
	g.AddEdge("C", "D")

	order, err := g.TopoSort()
	if err != nil {
		t.Fatalf("topo sort failed: %v", err)
	}
	if len(order) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(order))
	}

	// 验证 D 在 A、B、C 之后
	posD := indexOf(order, "D")
	posA := indexOf(order, "A")
	posB := indexOf(order, "B")
	posC := indexOf(order, "C")

	if posA >= posD || posB >= posD || posC >= posD {
		t.Errorf("topo order invalid: D should come after A, B, C. Got: %v", order)
	}
}

func TestDependencyGraph_TopoSort_Cycle(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddEdge("A", "B")
	// 强制制造一个循环（绕过 AddEdge 检测）
	g.edges["B"] = append(g.edges["B"], "A")
	// 但 reverseEdges 也要补全
	g.reverseEdges["A"] = append(g.reverseEdges["A"], "B")

	_, err := g.TopoSort()
	if err == nil {
		t.Error("expected error on cyclic graph")
	}
}

func TestDependencyGraph_Concurrent(t *testing.T) {
	g := NewDependencyGraph()
	for i := 0; i < 100; i++ {
		g.AddNode(string(rune('A' + i%26)))
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				from := string(rune('A' + id%26))
				to := string(rune('A' + j%26))
				_ = g.AddEdge(from, to)
			}
		}(i)
	}
	wg.Wait()
}

func indexOf(s []string, target string) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}
