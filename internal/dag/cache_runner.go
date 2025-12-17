package dag

import (
	"context"
	"fmt"

	"scriptweaver/internal/core"
)

// NodeResult is the deterministic outcome of executing (or replaying) a single node.
//
// This is used by the DAG executor to:
//   - commit the correct terminal state (COMPLETED/FAILED/CACHED)
//   - record stable per-node results in GraphResult
type NodeResult struct {
	Hash core.TaskHash

	Stdout   []byte
	Stderr   []byte
	ExitCode int

	FromCache         bool
	ArtifactsRestored int
}

// CacheAwareRunner adapts the Sprint-00 core.Runner to the DAG executor.
//
// It is responsible for:
//   - computing TaskHash
//   - checking cache
//   - replaying artifacts on hits
//   - executing and caching on misses
//
// Determinism is inherited from core.Runner's hashing, input resolution,
// and bit-for-bit replay.
type CacheAwareRunner struct {
	Runner *core.Runner
}

func NewCacheAwareRunner(r *core.Runner) (*CacheAwareRunner, error) {
	if r == nil {
		return nil, fmt.Errorf("nil core runner")
	}
	return &CacheAwareRunner{Runner: r}, nil
}

func (r *CacheAwareRunner) Run(ctx context.Context, task core.Task) (*NodeResult, error) {
	res, err := r.Runner.Run(ctx, &task)
	if err != nil {
		return nil, err
	}
	return &NodeResult{
		Hash:              res.Hash,
		Stdout:            res.Stdout,
		Stderr:            res.Stderr,
		ExitCode:          res.ExitCode,
		FromCache:         res.FromCache,
		ArtifactsRestored: res.ArtifactsRestored,
	}, nil
}

func (r *CacheAwareRunner) Probe(ctx context.Context, task core.Task) (*NodeResult, bool, error) {
	if r == nil || r.Runner == nil {
		return nil, false, fmt.Errorf("nil core runner")
	}
	if task.Name == "" {
		return nil, false, fmt.Errorf("task name is required")
	}
	if task.Run == "" {
		return nil, false, fmt.Errorf("task run command is required")
	}

	inputSet, err := r.Runner.Resolver.Resolve(task.Inputs)
	if err != nil {
		return nil, false, fmt.Errorf("resolving inputs: %w", err)
	}

	hashInput := core.HashInput{
		Inputs:     inputSet,
		Command:    task.Run,
		Env:        task.Env,
		Outputs:    task.Outputs,
		WorkingDir: r.Runner.WorkingDir,
	}
	hash := r.Runner.Hasher.ComputeHash(hashInput)

	exists, err := r.Runner.Cache.Has(hash)
	if err != nil {
		return nil, false, fmt.Errorf("checking cache: %w", err)
	}
	if !exists {
		return nil, false, nil
	}

	entry, err := r.Runner.Cache.Get(hash)
	if err != nil {
		return nil, false, fmt.Errorf("retrieving cache entry: %w", err)
	}
	if entry == nil {
		return nil, false, fmt.Errorf("cache entry disappeared")
	}

	replayResult, err := r.Runner.Replayer.Replay(entry)
	if err != nil {
		return nil, false, fmt.Errorf("replaying cached result: %w", err)
	}

	return &NodeResult{
		Hash:              hash,
		Stdout:            replayResult.Stdout,
		Stderr:            replayResult.Stderr,
		ExitCode:          replayResult.ExitCode,
		FromCache:         true,
		ArtifactsRestored: replayResult.ArtifactsRestored,
	}, true, nil
}
