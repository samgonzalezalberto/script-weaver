package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
)

func TestExecute_ResumeOnly_FailsWhenNoEligiblePreviousRun(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	// Minimal valid graph.
	tasks := []core.Task{{
		Name:    "A",
		Inputs:  []string{},
		Run:     "true",
		Outputs: []string{},
	}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeResumeOnly,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	_, err := Execute(context.Background(), inv)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_Incremental_ReusesCheckpointedWorkAfterFailure(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	// A writes a file (cached). B fails.
	tasks := []core.Task{
		{
			Name:    "A",
			Inputs:  []string{},
			Run:     "mkdir -p out && echo hello > out/a.txt",
			Outputs: []string{"out/a.txt"},
		},
		{
			Name:   "B",
			Inputs: []string{"out/a.txt"},
			Run:    "exit 7",
		},
	}
	edges := []dag.Edge{{From: "A", To: "B"}}
	writeGraphJSON(t, graphPath, tasks, edges)

	inv1 := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	res1, err := Execute(context.Background(), inv1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.ExitCode != ExitGraphFailure {
		t.Fatalf("expected graph failure exit, got %d", res1.ExitCode)
	}

	// Second run should reuse cache for A (planned reuse cache), and still fail on B.
	res2, err := Execute(context.Background(), inv1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.ExitCode != ExitGraphFailure {
		t.Fatalf("expected graph failure exit, got %d", res2.ExitCode)
	}
	if res2.GraphResult == nil {
		t.Fatalf("expected graph result")
	}
	// Validate that trace contains a TaskCached event for A (planned reuse cache path).
	var tj struct {
		Events []struct {
			Kind   string `json:"kind"`
			TaskID string `json:"taskId"`
		} `json:"events"`
	}
	if err := json.Unmarshal(res2.GraphResult.TraceBytes, &tj); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	found := false
	for _, e := range tj.Events {
		if e.TaskID == "A" && e.Kind == "TaskCached" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected TaskCached event for A")
	}
}
