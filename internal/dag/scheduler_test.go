package dag

import (
	"reflect"
	"testing"

	"scriptweaver/internal/core"
)

func TestScheduler_ReadyTasks_SortedByDepthThenName(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
			{Name: "D", Inputs: []string{"d"}, Run: "run-d"},
		},
		[]Edge{{From: "A", To: "C"}, {From: "B", To: "D"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A and B completed => C and D become ready. Both are depth 1, so lexical by name.
	state := ExecutionState{
		"A": TaskCompleted,
		"B": TaskCompleted,
		"C": TaskPending,
		"D": TaskPending,
	}

	got := GetReadyTasks(g, state)
	want := []string{"C", "D"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ready list mismatch: got %v want %v", got, want)
	}
}

func TestScheduler_ReadyTasks_RootsLexicalOrder(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state := ExecutionState{
		"A": TaskPending,
		"B": TaskPending,
		"C": TaskPending,
	}

	got := GetReadyTasks(g, state)
	want := []string{"A", "B", "C"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ready list mismatch: got %v want %v", got, want)
	}
}

func TestScheduler_DiamondConvergence_WaitsForAllParents(t *testing.T) {
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
		t.Fatalf("unexpected error: %v", err)
	}

	// After A completes, B and C are ready, D is not.
	state := ExecutionState{
		"A": TaskCompleted,
		"B": TaskPending,
		"C": TaskPending,
		"D": TaskPending,
	}
	got := GetReadyTasks(g, state)
	if !reflect.DeepEqual(got, []string{"B", "C"}) {
		t.Fatalf("unexpected ready list after A completed: %v", got)
	}

	// After B completes but C still pending, D must still not be ready.
	state["B"] = TaskCompleted
	got = GetReadyTasks(g, state)
	if !reflect.DeepEqual(got, []string{"C"}) {
		t.Fatalf("unexpected ready list after B completed: %v", got)
	}

	// After C is cached (equivalent to completed), D becomes ready.
	state["C"] = TaskCached
	got = GetReadyTasks(g, state)
	if !reflect.DeepEqual(got, []string{"D"}) {
		t.Fatalf("unexpected ready list after C cached: %v", got)
	}
}
