package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
)

type panicExecutor struct{}

func (panicExecutor) Run(context.Context, *dag.TaskGraph, dag.TaskRunner) (*dag.GraphResult, error) {
	panic("boom")
}

func writeGraphJSON(t *testing.T, path string, tasks []core.Task, edges []dag.Edge) {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"tasks": tasks,
		"edges": edges,
	})
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func TestExecute_OverwritePolicy_RemovesStaleFiles(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "stale.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	tasks := []core.Task{{
		Name:    "t1",
		Inputs:  []string{},
		Run:     "mkdir -p out && echo fresh > out/new.txt",
		Outputs: []string{"out/new.txt"},
	}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeClean,
	}

	res, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != ExitSuccess {
		t.Fatalf("expected exit %d got %d", ExitSuccess, res.ExitCode)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale file removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "new.txt")); err != nil {
		t.Fatalf("expected new output exists: %v", err)
	}
}

func TestExecute_ExitCodeGraphFailure(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	tasks := []core.Task{{
		Name:   "t1",
		Inputs: []string{},
		Run:    "exit 7",
	}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeClean,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	res, err := Execute(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != ExitGraphFailure {
		t.Fatalf("expected exit %d got %d", ExitGraphFailure, res.ExitCode)
	}
	if _, err := os.Stat(tracePath); err != nil {
		t.Fatalf("expected trace file exists: %v", err)
	}
}

func TestExecute_ConfigError_WhenOutputDirIsFile(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputFile := filepath.Join(workDir, "out")
	if err := os.WriteFile(outputFile, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write out file: %v", err)
	}

	tasks := []core.Task{{Name: "t1", Run: "true"}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputFile,
		ExecutionMode: ExecutionModeClean,
	}

	res, err := Execute(context.Background(), inv)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != ExitConfigError {
		t.Fatalf("expected exit %d got %d", ExitConfigError, res.ExitCode)
	}
}

func TestExecute_Panic_ExitCodeInternalAndTraceFinalized(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outputDir := filepath.Join(workDir, "out")
	tracePath := filepath.Join(workDir, "trace.json")

	tasks := []core.Task{{Name: "t1", Run: "true"}}
	writeGraphJSON(t, graphPath, tasks, nil)

	inv := CLIInvocation{
		WorkDir:       workDir,
		GraphPath:     graphPath,
		CacheDir:      filepath.Join(workDir, "cache"),
		OutputDir:     outputDir,
		ExecutionMode: ExecutionModeClean,
		Trace:         TraceConfig{Enabled: true, Path: tracePath},
	}

	res, err := ExecuteWithExecutor(context.Background(), inv, panicExecutor{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != ExitInternalError {
		t.Fatalf("expected exit %d got %d", ExitInternalError, res.ExitCode)
	}

	b, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("expected trace file exists: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("expected trace JSON: %v", err)
	}
	if decoded["graphHash"] == "" {
		t.Fatalf("expected graphHash in trace")
	}
}
