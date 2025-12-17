package dag

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"scriptweaver/internal/core"
	"scriptweaver/internal/incremental"
)

// Mixed Cached and Executed Runs:
// Upstream A is ReuseCache (restored), downstream B is Execute and consumes A's artifact.
func TestExecutorSerial_IncrementalPlan_MixedCachedAndExecuted_RestorationFeedsDownstream(t *testing.T) {
	workDir := t.TempDir()

	cache := core.NewMemoryCache()
	coreRunner := core.NewRunner(workDir, cache)
	cacheRunner, err := NewCacheAwareRunner(coreRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, err := NewTaskGraph(
		[]core.Task{
			{
				Name:    "A",
				Run:     "printf 'A1' > a.txt",
				Outputs: []string{"a.txt"},
			},
			{
				Name:    "B",
				Inputs:  []string{"a.txt"},
				Run:     `IFS= read -r x < a.txt; printf '%sB' "$x" > b.txt`,
				Outputs: []string{"b.txt"},
			},
		},
		[]Edge{{From: "A", To: "B"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run 1: populate cache (both execute).
	exec1, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res1, err := exec1.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.FinalState["A"] != TaskCompleted || res1.FinalState["B"] != TaskCompleted {
		t.Fatalf("expected completed on first run, got: %v", res1.FinalState)
	}
	if len(res1.TraceBytes) == 0 || res1.TraceHash == "" {
		t.Fatalf("expected trace bytes/hash on run 1")
	}

	// Delete artifacts to force restoration + execution to prove correctness.
	if err := os.Remove(filepath.Join(workDir, "a.txt")); err != nil {
		t.Fatalf("removing a.txt: %v", err)
	}
	if err := os.Remove(filepath.Join(workDir, "b.txt")); err != nil {
		t.Fatalf("removing b.txt: %v", err)
	}

	// Plan: A reused from cache, B executed.
	plan := &incremental.IncrementalPlan{
		Order: []string{"A", "B"},
		Decisions: map[string]incremental.NodeExecutionDecision{
			"A": incremental.DecisionReuseCache,
			"B": incremental.DecisionExecute,
		},
	}

	exec2, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exec2.Plan = plan

	res2, err := exec2.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.FinalState["A"] != TaskCompleted {
		t.Fatalf("expected A completed (restored), got %s", res2.FinalState["A"])
	}
	if res2.FinalState["B"] != TaskCompleted {
		t.Fatalf("expected B completed (executed), got %s", res2.FinalState["B"])
	}
	if len(res2.TraceBytes) == 0 || res2.TraceHash == "" {
		t.Fatalf("expected trace bytes/hash on run 2")
	}
	if string(res1.TraceBytes) == string(res2.TraceBytes) {
		t.Fatalf("expected incremental trace to differ from clean trace when events differ")
	}

	// Decode trace JSON to assert explicit "why not executed" and deterministic restore events.
	type decodedEvent struct {
		Kind        string   `json:"kind"`
		TaskID      string   `json:"taskId"`
		Reason      string   `json:"reason"`
		CauseTaskID string   `json:"causeTaskId"`
		Artifacts   []string `json:"artifacts"`
	}
	type decodedTrace struct {
		GraphHash string         `json:"graphHash"`
		Events    []decodedEvent `json:"events"`
	}

	var tr decodedTrace
	if err := json.Unmarshal(res2.TraceBytes, &tr); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	if tr.GraphHash == "" {
		t.Fatalf("expected graphHash in trace")
	}

	// Expectations:
	// - A: TaskCached with reason PlannedReuseCache
	// - A: TaskArtifactsRestored with reason CacheRestore
	// - A: MUST NOT have TaskExecuted
	// - B: TaskExecuted with reason PlannedExecute
	seen := map[string]bool{}
	for _, e := range tr.Events {
		key := e.TaskID + ":" + e.Kind + ":" + e.Reason
		seen[key] = true
		if e.TaskID == "A" && e.Kind == "TaskExecuted" {
			t.Fatalf("expected A to be cached (not executed), but saw TaskExecuted")
		}
	}
	if !seen["A:TaskCached:PlannedReuseCache"] {
		t.Fatalf("missing expected cached reason event for A")
	}
	if !seen["A:TaskArtifactsRestored:CacheRestore"] {
		t.Fatalf("missing expected artifacts restored event for A")
	}
	if !seen["B:TaskExecuted:PlannedExecute"] {
		t.Fatalf("missing expected executed event for B")
	}

	// Verify B could consume A's restored artifact.
	b, err := os.ReadFile(filepath.Join(workDir, "b.txt"))
	if err != nil {
		t.Fatalf("reading b.txt: %v", err)
	}
	if string(b) != "A1B" {
		t.Fatalf("unexpected B output: %q", b)
	}
}
