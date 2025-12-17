package dag

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"scriptweaver/internal/core"
)

type fakeRunner struct {
	exit map[string]int
}

func (r *fakeRunner) Probe(_ context.Context, _ core.Task) (*NodeResult, bool, error) {
	return nil, false, nil
}

func (r *fakeRunner) Run(_ context.Context, task core.Task) (*NodeResult, error) {
	if task.Name == "" {
		return nil, fmt.Errorf("missing task name")
	}

	exitCode := 0
	if r.exit == nil {
		return &NodeResult{Hash: core.TaskHash("hash:" + task.Name), ExitCode: 0}, nil
	}
	if code, ok := r.exit[task.Name]; ok {
		exitCode = code
	}
	return &NodeResult{Hash: core.TaskHash("hash:" + task.Name), ExitCode: exitCode}, nil
}

func TestExecutorSerial_RespectsSchedulerOrderOnComplexGraph(t *testing.T) {
	// Graph:
	//   A -> C
	//   B -> D
	//   E (independent)
	//
	// Initially ready (depth 0): A, B, E => lexical A,B,E.
	// After A completes: C becomes ready (depth 1), but B and E (depth 0) must run first.
	// After B completes: D becomes ready (depth 1).
	// After E completes: C and D both depth 1 => lexical C then D.
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
			{Name: "D", Inputs: []string{"d"}, Run: "run-d"},
			{Name: "E", Inputs: []string{"e"}, Run: "run-e"},
		},
		[]Edge{{From: "A", To: "C"}, {From: "B", To: "D"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exec, err := NewExecutor(g, &fakeRunner{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, err := exec.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantOrder := []string{"A", "B", "E", "C", "D"}
	if !reflect.DeepEqual(res.ExecutionOrder, wantOrder) {
		t.Fatalf("execution order mismatch: got %v want %v", res.ExecutionOrder, wantOrder)
	}

	for _, name := range []string{"A", "B", "C", "D", "E"} {
		if res.FinalState[name] != TaskCompleted {
			t.Fatalf("expected %s COMPLETED, got %s", name, res.FinalState[name])
		}
	}
}

func TestExecutorSerial_FailurePropagatesAndContinuesIndependentWork(t *testing.T) {
	// Graph:
	//   A -> B -> C
	//   D (independent)
	//
	// A fails; B and C become SKIPPED; D still runs.
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

	exec, err := NewExecutor(g, &fakeRunner{exit: map[string]int{"A": 7}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, err := exec.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Depth 0 nodes are A and D => lexical A then D.
	if !reflect.DeepEqual(res.ExecutionOrder, []string{"A", "D"}) {
		t.Fatalf("unexpected execution order: %v", res.ExecutionOrder)
	}

	if res.FinalState["A"] != TaskFailed {
		t.Fatalf("expected A failed, got %s", res.FinalState["A"])
	}
	if res.FinalState["B"] != TaskSkipped {
		t.Fatalf("expected B skipped, got %s", res.FinalState["B"])
	}
	if res.FinalState["C"] != TaskSkipped {
		t.Fatalf("expected C skipped, got %s", res.FinalState["C"])
	}
	if res.FinalState["D"] != TaskCompleted {
		t.Fatalf("expected D completed, got %s", res.FinalState["D"])
	}
}
