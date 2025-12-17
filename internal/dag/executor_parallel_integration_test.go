package dag

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"scriptweaver/internal/core"
)

type sleepyCountingRunner struct {
	exit   map[string]int
	delay  map[string]time.Duration
	mu     sync.Mutex
	counts map[string]int
}

func (r *sleepyCountingRunner) Probe(_ context.Context, _ core.Task) (*NodeResult, bool, error) {
	return nil, false, nil
}

func (r *sleepyCountingRunner) Run(_ context.Context, task core.Task) (*NodeResult, error) {
	d := time.Duration(0)
	if r.delay != nil {
		d = r.delay[task.Name]
	}
	if d > 0 {
		time.Sleep(d)
	}
	// Encourage scheduler interleavings.
	runtime.Gosched()

	r.mu.Lock()
	if r.counts == nil {
		r.counts = map[string]int{}
	}
	r.counts[task.Name]++
	r.mu.Unlock()

	exitCode := 0
	if r.exit == nil {
		return &NodeResult{Hash: core.TaskHash("hash:" + task.Name), ExitCode: 0}, nil
	}
	if code, ok := r.exit[task.Name]; ok {
		exitCode = code
	}
	return &NodeResult{Hash: core.TaskHash("hash:" + task.Name), ExitCode: exitCode}, nil
}

func TestExecutorParallel_RespectsDeterministicOrder(t *testing.T) {
	// Same complex graph as the serial test.
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

	serialExec, err := NewExecutor(g, &fakeRunner{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	serialRes, err := serialExec.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runner := &sleepyCountingRunner{delay: map[string]time.Duration{"A": 2 * time.Millisecond, "B": 1 * time.Millisecond}}
	parExec, err := NewExecutor(g, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parRes, err := parExec.RunParallel(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parRes.GraphHash != serialRes.GraphHash {
		t.Fatalf("graph hash mismatch: %s vs %s", parRes.GraphHash, serialRes.GraphHash)
	}
	if !reflect.DeepEqual(parRes.FinalState, serialRes.FinalState) {
		t.Fatalf("final state mismatch: par=%v serial=%v", parRes.FinalState, serialRes.FinalState)
	}
	if !reflect.DeepEqual(parRes.ExecutionOrder, serialRes.ExecutionOrder) {
		t.Fatalf("execution order mismatch: par=%v serial=%v", parRes.ExecutionOrder, serialRes.ExecutionOrder)
	}
}

func TestExecutorParallel_StableAcrossRuns_100(t *testing.T) {
	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Inputs: []string{"a"}, Run: "run-a"},
			{Name: "B", Inputs: []string{"b"}, Run: "run-b"},
			{Name: "C", Inputs: []string{"c"}, Run: "run-c"},
			{Name: "D", Inputs: []string{"d"}, Run: "run-d"},
			{Name: "E", Inputs: []string{"e"}, Run: "run-e"},
			{Name: "F", Inputs: []string{"f"}, Run: "run-f"},
			{Name: "G", Inputs: []string{"g"}, Run: "run-g"},
		},
		[]Edge{
			{From: "A", To: "C"},
			{From: "A", To: "D"},
			{From: "B", To: "D"},
			{From: "C", To: "E"},
			{From: "D", To: "F"},
			{From: "E", To: "G"},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	delays := map[string]time.Duration{
		"A": 2 * time.Millisecond,
		"B": 1 * time.Millisecond,
		"C": 3 * time.Millisecond,
		"D": 1 * time.Millisecond,
		"E": 2 * time.Millisecond,
		"F": 1 * time.Millisecond,
		"G": 1 * time.Millisecond,
	}

	var baseline *GraphResult
	for i := 0; i < 100; i++ {
		exec, err := NewExecutor(g, &sleepyCountingRunner{delay: delays})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res, err := exec.RunParallel(context.Background(), 8)
		if err != nil {
			t.Fatalf("run %d unexpected error: %v", i, err)
		}

		if baseline == nil {
			baseline = res
			continue
		}
		if res.GraphHash != baseline.GraphHash {
			t.Fatalf("run %d graph hash mismatch", i)
		}
		if !reflect.DeepEqual(res.FinalState, baseline.FinalState) {
			t.Fatalf("run %d final state mismatch: %v vs %v", i, res.FinalState, baseline.FinalState)
		}
		if !reflect.DeepEqual(res.ExecutionOrder, baseline.ExecutionOrder) {
			t.Fatalf("run %d order mismatch: %v vs %v", i, res.ExecutionOrder, baseline.ExecutionOrder)
		}
	}
}

func TestExecutorParallel_StateTransitionIntegrity_NoDuplicates(t *testing.T) {
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

	runner := &sleepyCountingRunner{delay: map[string]time.Duration{"A": 2 * time.Millisecond, "B": 2 * time.Millisecond}}
	exec, err := NewExecutor(g, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, err := exec.RunParallel(context.Background(), 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for name, st := range res.FinalState {
		if st == TaskRunning {
			t.Fatalf("task %q left RUNNING", name)
		}
	}

	runner.mu.Lock()
	defer runner.mu.Unlock()
	for _, name := range []string{"A", "B", "C", "D"} {
		if runner.counts[name] != 1 {
			t.Fatalf("expected %q to execute once, got %d", name, runner.counts[name])
		}
	}
}
