package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	icl "scriptweaver/internal/cli"
	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
)

func writeGraphJSON(t *testing.T, path string, tasks []core.Task, edges []dag.Edge) {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"tasks": tasks,
		"edges": edges,
	})
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func TestDeterministicInvocation_IdenticalRunsIdenticalArtifacts(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outDirRel := "out"
	traceRel := "trace.json"

	writeGraphJSON(t, graphPath,
		[]core.Task{{
			Name:    "t1",
			Run:     "mkdir -p out && echo hello > out/result.txt",
			Outputs: []string{"out/result.txt"},
		}},
		nil,
	)

	args := []string{
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", outDirRel,
		"--mode", "clean",
		"--trace", traceRel,
	}

	res1, err1 := icl.Run(context.Background(), args)
	if err1 != nil {
		t.Fatalf("run1 err: %v", err1)
	}
	if res1.ExitCode != icl.ExitSuccess {
		t.Fatalf("run1 exit: %d", res1.ExitCode)
	}
	outPath := filepath.Join(workDir, outDirRel, "result.txt")
	tracePath := filepath.Join(workDir, traceRel)
	out1 := readFile(t, outPath)
	tr1 := readFile(t, tracePath)

	res2, err2 := icl.Run(context.Background(), args)
	if err2 != nil {
		t.Fatalf("run2 err: %v", err2)
	}
	if res2.ExitCode != icl.ExitSuccess {
		t.Fatalf("run2 exit: %d", res2.ExitCode)
	}
	out2 := readFile(t, outPath)
	tr2 := readFile(t, tracePath)

	if string(out1) != string(out2) {
		t.Fatalf("artifact differs across identical runs")
	}
	if string(tr1) != string(tr2) {
		t.Fatalf("trace differs across identical runs")
	}
}

func TestPathResolution_RelativePathsResolveAgainstWorkDir(t *testing.T) {
	workDir := t.TempDir()
	otherCwd := t.TempDir()

	graphPath := filepath.Join(workDir, "graphs", "g.json")
	writeGraphJSON(t, graphPath,
		[]core.Task{{
			Name:    "t1",
			Run:     "mkdir -p out && echo ok > out/x.txt",
			Outputs: []string{"out/x.txt"},
		}},
		nil,
	)

	oldCwd, _ := os.Getwd()
	_ = os.Chdir(otherCwd)
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	args := []string{
		"--workdir", workDir,
		"--graph", "graphs/g.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
		"--trace", "traces/t.json",
	}

	res, err := icl.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.ExitCode != icl.ExitSuccess {
		t.Fatalf("exit: %d", res.ExitCode)
	}

	if _, err := os.Stat(filepath.Join(workDir, "out", "x.txt")); err != nil {
		t.Fatalf("expected output under workdir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "traces", "t.json")); err != nil {
		t.Fatalf("expected trace under workdir: %v", err)
	}
}

func TestExitCodeStability_FailingGraphIsStable(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	writeGraphJSON(t, graphPath,
		[]core.Task{{Name: "t1", Run: "exit 9"}},
		nil,
	)

	args := []string{
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
	}

	res1, _ := icl.Run(context.Background(), args)
	res2, _ := icl.Run(context.Background(), args)
	if res1.ExitCode != icl.ExitGraphFailure || res2.ExitCode != icl.ExitGraphFailure {
		t.Fatalf("expected stable graph failure exit code; got %d and %d", res1.ExitCode, res2.ExitCode)
	}
}

func TestCachePersistence_IncrementalReuseIsDeterministicAndTraceable(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	tracePath := filepath.Join(workDir, "trace.json")

	// counter.txt is NOT an output; it will only change if the task re-executes.
	writeGraphJSON(t, graphPath,
		[]core.Task{{
			Name:    "t1",
			Run:     "if [ -f counter.txt ]; then echo X >> counter.txt; else echo X > counter.txt; fi; mkdir -p out && echo artifact > out/out.txt",
			Outputs: []string{"out/out.txt"},
		}},
		nil,
	)

	args := []string{
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "incremental",
		"--trace", "trace.json",
	}

	res1, err := icl.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("run1 err: %v", err)
	}
	if res1.ExitCode != icl.ExitSuccess {
		t.Fatalf("run1 exit: %d", res1.ExitCode)
	}
	c1 := strings.TrimSpace(string(readFile(t, filepath.Join(workDir, "counter.txt"))))
	if c1 != "X" {
		t.Fatalf("expected counter created once, got %q", c1)
	}

	res2, err := icl.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("run2 err: %v", err)
	}
	if res2.ExitCode != icl.ExitSuccess {
		t.Fatalf("run2 exit: %d", res2.ExitCode)
	}
	c2 := strings.TrimSpace(string(readFile(t, filepath.Join(workDir, "counter.txt"))))
	if c2 != "X" {
		t.Fatalf("expected task not to re-execute (counter unchanged), got %q", c2)
	}
	if _, err := os.Stat(filepath.Join(workDir, "out", "out.txt")); err != nil {
		t.Fatalf("expected output restored from cache: %v", err)
	}

	// Trace should be valid JSON and include a TaskCached event on the second run.
	b := readFile(t, tracePath)
	var tr struct {
		GraphHash string `json:"graphHash"`
		Events    []struct {
			Kind string `json:"kind"`
		} `json:"events"`
	}
	if err := json.Unmarshal(b, &tr); err != nil {
		t.Fatalf("trace json invalid: %v", err)
	}
	if tr.GraphHash == "" {
		t.Fatalf("missing graphHash")
	}
	hasCached := false
	for _, e := range tr.Events {
		if e.Kind == "TaskCached" {
			hasCached = true
			break
		}
	}
	if !hasCached {
		t.Fatalf("expected TaskCached event in trace")
	}
}

func TestTraceEmission_EnabledProducesDeterministicCanonicalTrace(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	tracePath := filepath.Join(workDir, "trace.json")

	writeGraphJSON(t, graphPath,
		[]core.Task{{
			Name:    "t1",
			Run:     "mkdir -p out && echo z > out/z.txt",
			Outputs: []string{"out/z.txt"},
		}},
		nil,
	)

	args := []string{
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
		"--trace", "trace.json",
	}

	res1, err := icl.Run(context.Background(), args)
	if err != nil || res1.ExitCode != icl.ExitSuccess {
		t.Fatalf("run1 failed: exit=%d err=%v", res1.ExitCode, err)
	}
	b1 := readFile(t, tracePath)

	res2, err := icl.Run(context.Background(), args)
	if err != nil || res2.ExitCode != icl.ExitSuccess {
		t.Fatalf("run2 failed: exit=%d err=%v", res2.ExitCode, err)
	}
	b2 := readFile(t, tracePath)

	if string(b1) != string(b2) {
		t.Fatalf("expected deterministic trace bytes")
	}
	var decoded map[string]any
	if err := json.Unmarshal(b1, &decoded); err != nil {
		t.Fatalf("trace not valid json: %v", err)
	}
	if decoded["graphHash"] == "" {
		t.Fatalf("trace missing graphHash")
	}
}

func TestInvalidInvocation_DeterministicAndExplainable(t *testing.T) {
	workDir := t.TempDir()

	args := []string{
		"--workdir", workDir,
		"--cache-dir", "cache",
		"--output-dir", "out",
	}
	res1, err1 := icl.Run(context.Background(), args)
	res2, err2 := icl.Run(context.Background(), args)

	if res1.ExitCode != icl.ExitInvalidInvocation || res2.ExitCode != icl.ExitInvalidInvocation {
		t.Fatalf("expected exit 2, got %d and %d", res1.ExitCode, res2.ExitCode)
	}
	if err1 == nil || err2 == nil {
		t.Fatalf("expected errors")
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("expected deterministic error message")
	}
}

func TestOutputDeterminism_StaleFilesRemoved(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outDir := filepath.Join(workDir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "stale.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	writeGraphJSON(t, graphPath,
		[]core.Task{{
			Name:    "t1",
			Run:     "mkdir -p out && echo fresh > out/new.txt",
			Outputs: []string{"out/new.txt"},
		}},
		nil,
	)

	args := []string{
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
	}

	res, err := icl.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.ExitCode != icl.ExitSuccess {
		t.Fatalf("exit: %d", res.ExitCode)
	}
	if _, err := os.Stat(filepath.Join(outDir, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale removed")
	}
	if _, err := os.Stat(filepath.Join(outDir, "new.txt")); err != nil {
		t.Fatalf("expected new present: %v", err)
	}
}

func TestWriteFailure_ReadOnlyOutputDir_ReturnsExit3(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	outDir := filepath.Join(workDir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "stale.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}
	if err := os.Chmod(outDir, 0o555); err != nil {
		t.Fatalf("chmod out: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(outDir, 0o755) })

	writeGraphJSON(t, graphPath,
		[]core.Task{{Name: "t1", Run: "true"}},
		nil,
	)

	args := []string{
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
	}

	res, err := icl.Run(context.Background(), args)
	if res.ExitCode != icl.ExitConfigError {
		t.Fatalf("expected exit %d got %d (err=%v)", icl.ExitConfigError, res.ExitCode, err)
	}
	if err == nil {
		t.Fatalf("expected error")
	}
}
