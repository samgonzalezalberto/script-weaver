package dag

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"scriptweaver/internal/core"
	"scriptweaver/internal/incremental"
)

func TestTraceDeterminism_Parallelism1Vs8_TraceHashEquality(t *testing.T) {
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

	exec1, err := NewExecutor(g, &sleepyCountingRunner{delay: delays})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res1, err := exec1.RunParallel(context.Background(), 1)
	if err != nil {
		t.Fatalf("parallelism=1 unexpected error: %v", err)
	}

	exec8, err := NewExecutor(g, &sleepyCountingRunner{delay: delays})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res8, err := exec8.RunParallel(context.Background(), 8)
	if err != nil {
		t.Fatalf("parallelism=8 unexpected error: %v", err)
	}

	if res1.TraceHash != res8.TraceHash {
		t.Fatalf("trace hash mismatch: p1=%s p8=%s", res1.TraceHash, res8.TraceHash)
	}
	if !reflect.DeepEqual(res1.TraceBytes, res8.TraceBytes) {
		t.Fatalf("trace bytes mismatch: p1=%s p8=%s", string(res1.TraceBytes), string(res8.TraceBytes))
	}
}

func TestTraceDeterminism_IncrementalRun_RepeatedStable(t *testing.T) {
	workDir := t.TempDir()
	cache := core.NewMemoryCache()
	coreRunner := core.NewRunner(workDir, cache)
	cacheRunner, err := NewCacheAwareRunner(coreRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, err := NewTaskGraph(
		[]core.Task{
			{Name: "A", Run: "printf 'A1' > a.txt", Outputs: []string{"a.txt"}},
			{Name: "B", Inputs: []string{"a.txt"}, Run: `IFS= read -r x < a.txt; printf '%sB' "$x" > b.txt`, Outputs: []string{"b.txt"}},
		},
		[]Edge{{From: "A", To: "B"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run 1: populate cache.
	exec1, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = exec1.RunParallel(context.Background(), 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan: reuse cache for all tasks.
	plan := &incremental.IncrementalPlan{
		Order: []string{"A", "B"},
		Decisions: map[string]incremental.NodeExecutionDecision{
			"A": incremental.DecisionReuseCache,
			"B": incremental.DecisionReuseCache,
		},
	}

	// Run 2: incremental (reuse cache).
	exec2, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exec2.Plan = plan
	res2, err := exec2.RunParallel(context.Background(), 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate nothing; keep artifacts present. Run 3 should produce identical incremental trace.
	exec3, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exec3.Plan = plan
	res3, err := exec3.RunParallel(context.Background(), 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res2.TraceHash != res3.TraceHash {
		t.Fatalf("incremental trace hash mismatch: run2=%s run3=%s", res2.TraceHash, res3.TraceHash)
	}
	if !reflect.DeepEqual(res2.TraceBytes, res3.TraceBytes) {
		t.Fatalf("incremental trace bytes mismatch: run2=%s run3=%s", string(res2.TraceBytes), string(res3.TraceBytes))
	}

	// Ensure outputs still correct (sanity).
	b, err := os.ReadFile(filepath.Join(workDir, "b.txt"))
	if err != nil {
		t.Fatalf("reading b.txt: %v", err)
	}
	if string(b) != "A1B" {
		t.Fatalf("unexpected output: %q", b)
	}
}

func TestTraceDeterminism_TaskDelay_DoesNotAffectTraceHash(t *testing.T) {
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

	// Run with a delay injected into A.
	exec1, err := NewExecutor(g, &sleepyCountingRunner{delay: map[string]time.Duration{"A": 10 * time.Millisecond}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res1, err := exec1.RunParallel(context.Background(), 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run with the delay moved to B.
	exec2, err := NewExecutor(g, &sleepyCountingRunner{delay: map[string]time.Duration{"B": 10 * time.Millisecond}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res2, err := exec2.RunParallel(context.Background(), 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res1.TraceHash != res2.TraceHash {
		t.Fatalf("trace hash changed due to delay: %s vs %s", res1.TraceHash, res2.TraceHash)
	}
	if !reflect.DeepEqual(res1.TraceBytes, res2.TraceBytes) {
		t.Fatalf("trace bytes changed due to delay: %s vs %s", string(res1.TraceBytes), string(res2.TraceBytes))
	}
}
