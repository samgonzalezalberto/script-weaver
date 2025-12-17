package dag

import (
	"errors"
	"testing"

	"scriptweaver/internal/core"
)

func TestGraphConstruction_SingleNode(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{{Name: "A", Inputs: []string{"in.txt"}, Run: "echo hi"}},
		nil,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if g == nil {
		t.Fatalf("expected graph")
	}
	if g.Hash() == "" {
		t.Fatalf("expected non-empty graph hash")
	}
	if got := g.TopologicalOrder(); len(got) != 1 || got[0] != "A" {
		t.Fatalf("unexpected topo order: %v", got)
	}
}

func TestGraphConstruction_MultipleIndependentNodes(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	order := g.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %v", order)
	}
	// Deterministic: should be stable across runs (canonical order).
	seen := map[string]bool{}
	for _, n := range order {
		seen[n] = true
	}
	for _, n := range []string{"A", "B", "C"} {
		if !seen[n] {
			t.Fatalf("missing node %q in topo order: %v", n, order)
		}
	}
}

func TestGraphConstruction_DependencyChain(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
		},
		[]Edge{{From: "A", To: "B"}, {From: "B", To: "C"}},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	order := g.TopologicalOrder()
	pos := map[string]int{}
	for i, n := range order {
		pos[n] = i
	}
	if !(pos["A"] < pos["B"] && pos["B"] < pos["C"]) {
		t.Fatalf("expected A < B < C, got %v", order)
	}
}

func TestGraphConstruction_DiamondDependency(t *testing.T) {
	// A -> B, A -> C, B -> D, C -> D
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
			{Name: "D", Inputs: []string{"d"}, Run: "run-d"},
		},
		[]Edge{{From: "A", To: "B"}, {From: "A", To: "C"}, {From: "B", To: "D"}, {From: "C", To: "D"}},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	order := g.TopologicalOrder()
	pos := map[string]int{}
	for i, n := range order {
		pos[n] = i
	}
	if !(pos["A"] < pos["B"] && pos["A"] < pos["C"]) {
		t.Fatalf("expected A before B and C, got %v", order)
	}
	if !(pos["B"] < pos["D"] && pos["C"] < pos["D"]) {
		t.Fatalf("expected D after B and C, got %v", order)
	}

	edges := g.Edges()
	countToD := 0
	for _, e := range edges {
		if e.To == "D" {
			countToD++
		}
	}
	if countToD != 2 {
		t.Fatalf("expected D to have 2 incoming edges, got %d", countToD)
	}
}

func TestGraphHash_InvariantToInsertionOrder(t *testing.T) {
	tasks1 := []core.Task{
		{Name: "A", Inputs: []string{"b", "a"}, Run: "echo A", Env: map[string]string{"Z": "9", "A": "1"}},
		{Name: "B", Inputs: []string{"x"}, Run: "echo B"},
		{Name: "C", Inputs: []string{"y"}, Run: "echo C"},
	}
	edges1 := []Edge{{From: "A", To: "B"}, {From: "A", To: "C"}}

	g1, err := NewTaskGraph(tasks1, edges1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same structure, different insertion orders.
	tasks2 := []core.Task{
		{Name: "C", Inputs: []string{"y"}, Run: "echo C"},
		{Name: "B", Inputs: []string{"x"}, Run: "echo B"},
		{Name: "A", Inputs: []string{"a", "b"}, Run: "echo A", Env: map[string]string{"A": "1", "Z": "9"}},
	}
	edges2 := []Edge{{From: "A", To: "C"}, {From: "A", To: "B"}}

	g2, err := NewTaskGraph(tasks2, edges2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g1.Hash() != g2.Hash() {
		t.Fatalf("expected equal graph hashes, got %s vs %s", g1.Hash(), g2.Hash())
	}
}

func TestCycleDetection_SelfLoopRejected(t *testing.T) {
	_, err := NewTaskGraph(
		[]core.Task{{Name: "A", Inputs: []string{"a"}, Run: "run-a"}},
		[]Edge{{From: "A", To: "A"}},
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrInvalidGraph) {
		t.Fatalf("expected invalid graph error, got %v", err)
	}
}

func TestCycleDetection_IndirectCycleRejected(t *testing.T) {
	_, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
		},
		[]Edge{{From: "A", To: "B"}, {From: "B", To: "C"}, {From: "C", To: "A"}},
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrCycleFound) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}
