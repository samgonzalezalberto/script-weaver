package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
	"scriptweaver/internal/trace"
)

// GraphExecutor is the minimal engine interface the CLI wires into.
//
// This allows the CLI to prove exit-code mapping (including panic) in tests
// without depending on specific executor internals.
type GraphExecutor interface {
	Run(ctx context.Context, graph *dag.TaskGraph, runner dag.TaskRunner) (*dag.GraphResult, error)
}

type defaultGraphExecutor struct{}

func (defaultGraphExecutor) Run(ctx context.Context, graph *dag.TaskGraph, runner dag.TaskRunner) (*dag.GraphResult, error) {
	exec, err := dag.NewExecutor(graph, runner)
	if err != nil {
		return nil, err
	}
	return exec.RunSerial(ctx)
}

type CLIResult struct {
	ExitCode   int
	GraphResult *dag.GraphResult
}

// Execute is the default entrypoint for running a canonical invocation.
func Execute(ctx context.Context, inv CLIInvocation) (CLIResult, error) {
	return ExecuteWithExecutor(ctx, inv, defaultGraphExecutor{})
}

// Execute maps a canonical CLIInvocation to engine execution.
//
// Responsibilities:
//   - Prepare OutputDir using the Overwrite policy (no stale files).
//   - Select cache strategy based on ExecutionMode.
//   - Initialize trace output before execution and finalize after execution,
//     even on panic/failure.
//   - Translate engine outcomes to semantic exit codes.
func ExecuteWithExecutor(ctx context.Context, inv CLIInvocation, executor GraphExecutor) (res CLIResult, execErr error) {
	res.ExitCode = ExitInternalError
	if executor == nil {
		return res, fmt.Errorf("nil executor")
	}

	graph, graphHash, err := loadGraphAndHash(inv.GraphPath)
	if err != nil {
		res.ExitCode = ExitConfigError
		return res, err
	}

	traceWriter, err := newTraceWriter(inv, graphHash)
	if err != nil {
		res.ExitCode = ExitConfigError
		return res, err
	}
	defer func() {
		// Always finalize trace output deterministically.
		_ = traceWriter.Finalize(res.GraphResult)
	}()

	if err := prepareOutputDir(inv.OutputDir); err != nil {
		res.ExitCode = ExitConfigError
		return res, err
	}

	cache, err := cacheForMode(inv.ExecutionMode, inv.CacheDir)
	if err != nil {
		res.ExitCode = ExitConfigError
		return res, err
	}

	runner := core.NewRunner(inv.WorkDir, cache)
	cacheRunner, err := dag.NewCacheAwareRunner(runner)
	if err != nil {
		res.ExitCode = ExitInternalError
		return res, err
	}

	defer func() {
		if r := recover(); r != nil {
			res.ExitCode = ExitInternalError
			res.GraphResult = nil
			execErr = fmt.Errorf("panic: %v", r)
		}
	}()

	gr, err := executor.Run(ctx, graph, cacheRunner)
	if err != nil {
		res.ExitCode = ExitInternalError
		return res, err
	}
	res.GraphResult = gr
	res.ExitCode = translateGraphResultToExitCode(gr)
	return res, nil
}

func translateGraphResultToExitCode(gr *dag.GraphResult) int {
	if gr == nil {
		return ExitInternalError
	}
	for _, st := range gr.FinalState {
		if st == dag.TaskFailed {
			return ExitGraphFailure
		}
	}
	return ExitSuccess
}

func cacheForMode(mode ExecutionMode, cacheDir string) (core.Cache, error) {
	switch mode {
	case ExecutionModeIncremental:
		if cacheDir == "" {
			return nil, fmt.Errorf("cache dir is empty")
		}
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("create cache dir: %w", err)
		}
		return core.NewFileCache(cacheDir), nil
	case ExecutionModeClean:
		return noCache{}, nil
	default:
		return nil, fmt.Errorf("unknown execution mode: %q", mode)
	}
}

type noCache struct{}

func (noCache) Has(core.TaskHash) (bool, error) { return false, nil }
func (noCache) Get(core.TaskHash) (*core.CacheEntry, error) { return nil, nil }
func (noCache) Put(*core.CacheEntry) error { return nil }

func prepareOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output dir is empty")
	}
	clean := filepath.Clean(dir)
	if clean == "/" {
		return fmt.Errorf("refusing to operate on output dir '/' ")
	}
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(clean, 0o755)
		}
		return fmt.Errorf("stat output dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output dir is not a directory: %s", clean)
	}
	entries, err := os.ReadDir(clean)
	if err != nil {
		return fmt.Errorf("read output dir: %w", err)
	}
	for _, e := range entries {
		p := filepath.Join(clean, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("clear output dir: %w", err)
		}
	}
	return nil
}

func loadGraphAndHash(path string) (*dag.TaskGraph, string, error) {
	g, err := LoadGraphFromFile(path)
	if err != nil {
		return nil, "", err
	}
	return g, g.Hash().String(), nil
}

type traceFileWriter struct {
	enabled bool
	path    string
	graphHash string
}

func newTraceWriter(inv CLIInvocation, graphHash string) (*traceFileWriter, error) {
	if !inv.Trace.Enabled {
		return &traceFileWriter{enabled: false}, nil
	}
	if inv.Trace.Path == "" {
		return nil, fmt.Errorf("trace enabled but path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(inv.Trace.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create trace dir: %w", err)
	}
	// Create an empty trace file eagerly so the destination is reserved and
	// so that even a panic results in a deterministic artifact.
	w := &traceFileWriter{enabled: true, path: inv.Trace.Path, graphHash: graphHash}
	return w, w.writeBytes(trace.ExecutionTrace{GraphHash: graphHash, Events: nil})
}

func (w *traceFileWriter) Finalize(gr *dag.GraphResult) error {
	if w == nil || !w.enabled {
		return nil
	}
	if gr != nil && len(gr.TraceBytes) > 0 {
		return writeFileAtomic(w.path, gr.TraceBytes, 0o644)
	}
	// If we don't have trace bytes (e.g., internal error or panic), still emit a valid
	// empty trace for this graph.
	return w.writeBytes(trace.ExecutionTrace{GraphHash: w.graphHash, Events: nil})
}

func (w *traceFileWriter) writeBytes(t trace.ExecutionTrace) error {
	b, err := t.CanonicalJSON()
	if err != nil {
		return err
	}
	return writeFileAtomic(w.path, b, 0o644)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	_ = tmp.Sync() // best-effort durability
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
