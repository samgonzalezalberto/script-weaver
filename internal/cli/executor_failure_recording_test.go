package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"scriptweaver/internal/dag"
)

// stubExecutor returns a GraphResult containing a deterministic node failure.
type stubExecutor struct{}

func (stubExecutor) Run(ctx context.Context, graph *dag.TaskGraph, runner dag.TaskRunner) (*dag.GraphResult, error) {
	return &dag.GraphResult{FinalState: map[string]dag.TaskState{"A": dag.TaskFailed}}, nil
}

func TestFailureRecording_WritesFailureJSON_OnNodeFailure(t *testing.T) {
	work := t.TempDir()

	// Minimal workspace dirs expected by CLI.
	if err := os.MkdirAll(filepath.Join(work, ".scriptweaver"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	inv := CLIInvocation{
		GraphPath:     filepath.Join(work, "graph.json"),
		WorkDir:       work,
		CacheDir:      filepath.Join(work, "cache"),
		OutputDir:     filepath.Join(work, "out"),
		ExecutionMode: ExecutionModeIncremental,
		Trace:         TraceConfig{Enabled: false},
	}

	// Minimal valid graph file to pass graph loading.
	graphJSON := `{
	  "tasks": [
	    {"name": "A", "inputs": [], "run": ""}
	  ],
	  "edges": []
	}`
	if err := os.WriteFile(inv.GraphPath, []byte(graphJSON), 0o644); err != nil {
		t.Fatalf("WriteFile graph: %v", err)
	}

	res, err := ExecuteWithExecutor(context.Background(), inv, stubExecutor{})
	if err != nil {
		t.Fatalf("ExecuteWithExecutor: %v", err)
	}
	if res.ExitCode != ExitGraphFailure {
		t.Fatalf("expected ExitGraphFailure got %d", res.ExitCode)
	}
	// We don't know the run id, but a failure should have been recorded under .scriptweaver/runs.
	runsDir := filepath.Join(work, ".scriptweaver", "runs")
	entries, readErr := os.ReadDir(runsDir)
	if readErr != nil {
		t.Fatalf("ReadDir runs: %v", readErr)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least one run dir")
	}

	found := false
	for _, e := range entries {
		p := filepath.Join(runsDir, e.Name(), "failure.json")
		if _, statErr := os.Stat(p); statErr == nil {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected failure.json to exist in a run directory")
	}
}
