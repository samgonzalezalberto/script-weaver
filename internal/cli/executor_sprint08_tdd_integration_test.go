package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
	"scriptweaver/internal/recovery/state"
)

type traceDoc struct {
	Events []struct {
		Kind   string `json:"kind"`
		TaskID string `json:"taskId"`
		Reason string `json:"reason"`
	} `json:"events"`
}

func mustUnmarshalTrace(t *testing.T, traceBytes []byte) traceDoc {
	t.Helper()
	var td traceDoc
	if err := json.Unmarshal(traceBytes, &td); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	return td
}

func hasEvent(td traceDoc, taskID, kind string) bool {
	for _, e := range td.Events {
		if e.TaskID == taskID && e.Kind == kind {
			return true
		}
	}
	return false
}

func newRunIDFromDiff(t *testing.T, before, after []string) string {
	t.Helper()
	set := make(map[string]struct{}, len(before))
	for _, id := range before {
		set[id] = struct{}{}
	}
	var diff []string
	for _, id := range after {
		if _, ok := set[id]; !ok {
			diff = append(diff, id)
		}
	}
	if len(diff) != 1 {
		t.Fatalf("expected exactly one new run id, got %v", diff)
	}
	return diff[0]
}

func setRunStartTime(t *testing.T, st *state.Store, runID string, ts time.Time) {
	t.Helper()
	r, err := st.LoadRun(runID)
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	r.StartTime = ts
	if err := st.SaveRun(r); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
}

func TestSprint08_HappyResume_A_B_C_FailAtD_ResumeRunsD(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")
	flagPath := filepath.Join(workDir, "flag.txt")
	if err := os.WriteFile(flagPath, []byte("fail\n"), 0o644); err != nil {
		t.Fatalf("WriteFile flag: %v", err)
	}

	tasks := []core.Task{
		{Name: "A", Run: "mkdir -p out && echo a > out/a.txt", Outputs: []string{"out/a.txt"}},
		{Name: "B", Inputs: []string{"out/a.txt"}, Run: "cat out/a.txt > out/b.txt", Outputs: []string{"out/b.txt"}},
		{Name: "C", Inputs: []string{"out/b.txt"}, Run: "cat out/b.txt > out/c.txt", Outputs: []string{"out/c.txt"}},
		{
			Name:    "D",
			Inputs:  []string{"flag.txt", "out/c.txt"},
			Run:     "if grep -q fail flag.txt; then exit 7; fi; cat out/c.txt > out/d.txt",
			Outputs: []string{"out/d.txt"},
		},
	}
	edges := []dag.Edge{{From: "A", To: "B"}, {From: "B", To: "C"}, {From: "C", To: "D"}}
	writeGraphJSON(t, graphPath, tasks, edges)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	st, err := state.NewStore(workDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	before, _ := st.ListRunIDs()

	res1, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.ExitCode != ExitGraphFailure {
		t.Fatalf("expected graph failure, got %d", res1.ExitCode)
	}
	after1, _ := st.ListRunIDs()
	run1 := newRunIDFromDiff(t, before, after1)
	setRunStartTime(t, st, run1, time.Unix(1, 0).UTC())

	// Checkpoints must exist for A/B/C after failing at D.
	for _, n := range []string{"A", "B", "C"} {
		if _, err := st.LoadCheckpoint(run1, n); err != nil {
			t.Fatalf("expected checkpoint for %s: %v", n, err)
		}
	}

	// Allow D to succeed without changing the graph hash (input content change only).
	if err := os.WriteFile(flagPath, []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("WriteFile flag: %v", err)
	}

	before2 := after1
	res2, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.ExitCode != ExitSuccess {
		t.Fatalf("expected success, got %d", res2.ExitCode)
	}
	after2, _ := st.ListRunIDs()
	run2 := newRunIDFromDiff(t, before2, after2)
	setRunStartTime(t, st, run2, time.Unix(2, 0).UTC())

	if res2.GraphResult == nil {
		t.Fatalf("expected graph result")
	}
	td := mustUnmarshalTrace(t, res2.GraphResult.TraceBytes)
	for _, n := range []string{"A", "B", "C"} {
		if !hasEvent(td, n, "TaskCached") {
			t.Fatalf("expected TaskCached for %s", n)
		}
	}
	if !hasEvent(td, "D", "TaskExecuted") {
		t.Fatalf("expected TaskExecuted for D")
	}

	// Run linking: second run must point to the failed first run and increment retry_count.
	r2, err := st.LoadRun(run2)
	if err != nil {
		t.Fatalf("LoadRun run2: %v", err)
	}
	if r2.PreviousRunID == nil || *r2.PreviousRunID != run1 {
		t.Fatalf("expected previous_run_id=%q got %#v", run1, r2.PreviousRunID)
	}
	if r2.RetryCount != 1 {
		t.Fatalf("expected retry_count=1 got %d", r2.RetryCount)
	}
}

func TestSprint08_UpstreamInvalidation_InputChangeAtB_ForcesBAndDownstreamExecute(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")
	flagPath := filepath.Join(workDir, "flag.txt")
	bInputPath := filepath.Join(workDir, "b.txt")
	if err := os.WriteFile(flagPath, []byte("fail\n"), 0o644); err != nil {
		t.Fatalf("WriteFile flag: %v", err)
	}
	if err := os.WriteFile(bInputPath, []byte("one\n"), 0o644); err != nil {
		t.Fatalf("WriteFile b: %v", err)
	}

	tasks := []core.Task{
		{Name: "A", Run: "mkdir -p out && echo a > out/a.txt", Outputs: []string{"out/a.txt"}},
		{Name: "B", Inputs: []string{"out/a.txt", "b.txt"}, Run: "cat out/a.txt b.txt > out/b.txt", Outputs: []string{"out/b.txt"}},
		{Name: "C", Inputs: []string{"out/b.txt"}, Run: "cat out/b.txt > out/c.txt", Outputs: []string{"out/c.txt"}},
		{Name: "D", Inputs: []string{"flag.txt", "out/c.txt"}, Run: "if grep -q fail flag.txt; then exit 7; fi; cat out/c.txt > out/d.txt", Outputs: []string{"out/d.txt"}},
	}
	edges := []dag.Edge{{From: "A", To: "B"}, {From: "B", To: "C"}, {From: "C", To: "D"}}
	writeGraphJSON(t, graphPath, tasks, edges)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	st, _ := state.NewStore(workDir)
	before, _ := st.ListRunIDs()
	res1, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.ExitCode != ExitGraphFailure {
		t.Fatalf("expected graph failure, got %d", res1.ExitCode)
	}
	after1, _ := st.ListRunIDs()
	run1 := newRunIDFromDiff(t, before, after1)
	setRunStartTime(t, st, run1, time.Unix(1, 0).UTC())

	// Change B input and allow D to succeed.
	if err := os.WriteFile(bInputPath, []byte("two\n"), 0o644); err != nil {
		t.Fatalf("WriteFile b: %v", err)
	}
	if err := os.WriteFile(flagPath, []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("WriteFile flag: %v", err)
	}

	res2, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.ExitCode != ExitSuccess {
		t.Fatalf("expected success, got %d", res2.ExitCode)
	}
	if res2.GraphResult == nil {
		t.Fatalf("expected graph result")
	}
	td := mustUnmarshalTrace(t, res2.GraphResult.TraceBytes)
	if !hasEvent(td, "A", "TaskCached") {
		t.Fatalf("expected A cached")
	}
	if !hasEvent(td, "B", "TaskExecuted") {
		t.Fatalf("expected B executed")
	}
	if !hasEvent(td, "C", "TaskExecuted") {
		t.Fatalf("expected C executed")
	}
	if hasEvent(td, "C", "TaskCached") {
		t.Fatalf("expected C not cached")
	}
}

func TestSprint08_NonResumableFailure_GraphFailureRejectsResumeOnly(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	tasks := []core.Task{{Name: "A", Run: "true"}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv1 := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	st, _ := state.NewStore(workDir)
	before, _ := st.ListRunIDs()
	res, err := Execute(context.Background(), inv1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != ExitSuccess {
		t.Fatalf("expected success, got %d", res.ExitCode)
	}
	after, _ := st.ListRunIDs()
	run1 := newRunIDFromDiff(t, before, after)
	setRunStartTime(t, st, run1, time.Unix(1, 0).UTC())

	// Force a non-resumable GraphFailure for this run.
	if err := st.SaveFailure(run1, state.Failure{FailureClass: state.FailureClassGraph, ErrorCode: "SchemaViolation", ErrorMessage: "bad", Resumable: false}); err != nil {
		t.Fatalf("SaveFailure: %v", err)
	}

	inv2 := inv1
	inv2.ExecutionMode = ExecutionModeResumeOnly
	_, err = Execute(context.Background(), inv2)
	if err == nil {
		t.Fatalf("expected resume-only to be rejected")
	}
}

func TestSprint08_WorkspaceCorruption_DeletedCacheEntryRejectsResumeOnly(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	tasks := []core.Task{{Name: "A", Run: "mkdir -p out && echo hi > out/a.txt", Outputs: []string{"out/a.txt"}}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv1 := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	st, _ := state.NewStore(workDir)
	before, _ := st.ListRunIDs()
	res, err := Execute(context.Background(), inv1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != ExitSuccess {
		t.Fatalf("expected success, got %d", res.ExitCode)
	}
	after, _ := st.ListRunIDs()
	run1 := newRunIDFromDiff(t, before, after)
	setRunStartTime(t, st, run1, time.Unix(1, 0).UTC())

	// Simulate prior system failure so resume is attempted.
	if err := st.SaveFailure(run1, state.Failure{FailureClass: state.FailureClassSystem, ErrorCode: "Crash", ErrorMessage: "crash", Resumable: true}); err != nil {
		t.Fatalf("SaveFailure: %v", err)
	}
	cp, err := st.LoadCheckpoint(run1, "A")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	h := cp.CacheKeys[0]
	if len(h) < 2 {
		t.Fatalf("unexpected hash: %q", h)
	}
	meta := filepath.Join(inv1.CacheDir, h[:2], h, "metadata.json")
	if err := os.Remove(meta); err != nil {
		t.Fatalf("Remove metadata: %v", err)
	}

	inv2 := inv1
	inv2.ExecutionMode = ExecutionModeResumeOnly
	_, err = Execute(context.Background(), inv2)
	if err == nil {
		t.Fatalf("expected resume-only to be rejected due to corruption")
	}
}

func TestSprint08_CrashRecovery_PartialCheckpointsAllowResumeFromLastValid(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	// Build a simple chain A -> B -> C.
	tasks := []core.Task{
		{Name: "A", Run: "mkdir -p out && echo a > out/a.txt", Outputs: []string{"out/a.txt"}},
		{Name: "B", Inputs: []string{"out/a.txt"}, Run: "cat out/a.txt > out/b.txt", Outputs: []string{"out/b.txt"}},
		{Name: "C", Inputs: []string{"out/b.txt"}, Run: "cat out/b.txt > out/c.txt", Outputs: []string{"out/c.txt"}},
	}
	edges := []dag.Edge{{From: "A", To: "B"}, {From: "B", To: "C"}}
	writeGraphJSON(t, graphPath, tasks, edges)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	st, _ := state.NewStore(workDir)
	before, _ := st.ListRunIDs()
	res, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != ExitSuccess {
		t.Fatalf("expected success, got %d", res.ExitCode)
	}
	after, _ := st.ListRunIDs()
	seedRun := newRunIDFromDiff(t, before, after)

	// Simulate a crash run that only checkpointed A and B.
	seed, err := st.LoadRun(seedRun)
	if err != nil {
		t.Fatalf("LoadRun seed: %v", err)
	}
	crashRun := "crash"
	if err := st.SaveRun(state.Run{RunID: crashRun, GraphHash: seed.GraphHash, StartTime: time.Unix(1, 0).UTC(), Mode: state.ExecutionModeIncremental, RetryCount: 0, Status: "failed", PreviousRunID: nil}); err != nil {
		t.Fatalf("SaveRun crash: %v", err)
	}
	if err := st.SaveFailure(crashRun, state.Failure{FailureClass: state.FailureClassSystem, ErrorCode: "Crash", ErrorMessage: "crash", Resumable: true}); err != nil {
		t.Fatalf("SaveFailure crash: %v", err)
	}
	for _, n := range []string{"A", "B"} {
		cp, err := st.LoadCheckpoint(seedRun, n)
		if err != nil {
			t.Fatalf("LoadCheckpoint seed %s: %v", n, err)
		}
		if err := st.SaveCheckpoint(crashRun, cp); err != nil {
			t.Fatalf("SaveCheckpoint crash %s: %v", n, err)
		}
	}

	// Run again: should reuse cache for A and B, and execute C.
	res2, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.ExitCode != ExitSuccess {
		t.Fatalf("expected success, got %d", res2.ExitCode)
	}
	if res2.GraphResult == nil {
		t.Fatalf("expected graph result")
	}
	td := mustUnmarshalTrace(t, res2.GraphResult.TraceBytes)
	if !hasEvent(td, "A", "TaskCached") || !hasEvent(td, "B", "TaskCached") {
		t.Fatalf("expected A and B cached")
	}
	if !hasEvent(td, "C", "TaskExecuted") {
		t.Fatalf("expected C executed")
	}
}
