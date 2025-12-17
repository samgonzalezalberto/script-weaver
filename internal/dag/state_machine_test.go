package dag

import (
	"reflect"
	"testing"

	"scriptweaver/internal/core"
)

func TestStateMachine_Transitions_ValidAndInvalid(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{{Name: "A", Inputs: []string{"a"}, Run: "run-a"}},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = g

	state := ExecutionState{"A": TaskPending}

	if err := Transition(state, "A", TaskPending, TaskRunning); err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
	if err := Transition(state, "A", TaskRunning, TaskCompleted); err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}

	// Terminal -> RUNNING is forbidden.
	if err := Transition(state, "A", TaskCompleted, TaskRunning); err == nil {
		t.Fatalf("expected error")
	}

	// FAILED -> RUNNING is forbidden.
	state["A"] = TaskFailed
	if err := Transition(state, "A", TaskFailed, TaskRunning); err == nil {
		t.Fatalf("expected error")
	}

	// SKIPPED is terminal.
	state["A"] = TaskSkipped
	if err := Transition(state, "A", TaskSkipped, TaskRunning); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFailurePropagation_CascadeFailure_MarksDownstreamSkipped(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
			{Name: "D", Inputs: []string{"d"}, Run: "run-d"},
		},
		[]Edge{{From: "A", To: "B"}, {From: "B", To: "C"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state := ExecutionState{
		"A": TaskRunning,
		"B": TaskPending,
		"C": TaskPending,
		"D": TaskPending, // independent
	}

	if err := FailAndPropagate(g, state, "A"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state["A"] != TaskFailed {
		t.Fatalf("expected A failed, got %s", state["A"])
	}
	if state["B"] != TaskSkipped {
		t.Fatalf("expected B skipped, got %s", state["B"])
	}
	if state["C"] != TaskSkipped {
		t.Fatalf("expected C skipped, got %s", state["C"])
	}
	if state["D"] != TaskPending {
		t.Fatalf("expected D unchanged pending, got %s", state["D"])
	}

	// Scheduler gate: only independent root D should be ready now.
	got := GetReadyTasks(g, state)
	want := []string{"D"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ready mismatch: got %v want %v", got, want)
	}
}

func TestFailurePropagation_Diamond_DownstreamSkippedNotFailed(t *testing.T) {
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

	state := ExecutionState{
		"A": TaskRunning,
		"B": TaskPending,
		"C": TaskPending,
		"D": TaskPending,
	}

	if err := FailAndPropagate(g, state, "A"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state["B"] != TaskSkipped || state["C"] != TaskSkipped || state["D"] != TaskSkipped {
		t.Fatalf("expected B,C,D skipped; got B=%s C=%s D=%s", state["B"], state["C"], state["D"])
	}
	if state["D"] == TaskFailed {
		t.Fatalf("expected D skipped, not failed")
	}
}

func TestFailurePropagation_DetectsRunningDownstreamInvariantViolation(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
		},
		[]Edge{{From: "A", To: "B"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state := ExecutionState{
		"A": TaskRunning,
		"B": TaskRunning,
	}

	if err := FailAndPropagate(g, state, "A"); err == nil {
		t.Fatalf("expected error")
	}
}
