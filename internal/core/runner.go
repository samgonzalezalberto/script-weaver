// Package core defines the domain models for deterministic task execution.
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Runner orchestrates deterministic task execution with caching.
//
// The runner implements the full execution flow:
//  1. Resolve inputs and compute task hash
//  2. Check cache for existing result
//  3. If cached: replay (skip execution)
//  4. If not cached: execute, harvest artifacts, cache result
//
// From spec.md Failure Behavior:
//   - Failed executions (non-zero exit code) are cacheable.
//   - Failed tasks MUST NOT partially update artifacts.
type Runner struct {
	// WorkingDir is the task execution directory.
	WorkingDir string

	// Cache stores and retrieves execution results.
	Cache Cache

	// Executor runs tasks in isolated environments.
	Executor *Executor

	// Resolver expands input patterns to files.
	Resolver *InputResolver

	// Hasher computes deterministic task hashes.
	Hasher *TaskHasher

	// Harvester collects output artifacts.
	Harvester *Harvester

	// Replayer restores cached results.
	Replayer *Replayer

	// Normalizer for output normalization (optional).
	Normalizer OutputNormalizer
}

// NewRunner creates a Runner with the given working directory and cache.
func NewRunner(workingDir string, cache Cache) *Runner {
	return &Runner{
		WorkingDir: workingDir,
		Cache:      cache,
		Executor:   NewExecutor(workingDir),
		Resolver:   NewInputResolver(workingDir),
		Hasher:     NewTaskHasher(),
		Harvester:  NewHarvester(workingDir),
		Replayer:   NewReplayer(workingDir),
		Normalizer: nil,
	}
}

// NewRunnerWithNormalizer creates a Runner with output normalization.
func NewRunnerWithNormalizer(workingDir string, cache Cache, normalizer OutputNormalizer) *Runner {
	r := NewRunner(workingDir, cache)
	r.Normalizer = normalizer
	r.Harvester = NewHarvesterWithNormalizer(workingDir, normalizer)
	return r
}

// RunResult contains the result of running a task.
type RunResult struct {
	// Hash is the computed task hash.
	Hash TaskHash

	// Stdout is the task output.
	Stdout []byte

	// Stderr is the task error output.
	Stderr []byte

	// ExitCode is the process exit code.
	ExitCode int

	// FromCache indicates if the result was replayed from cache.
	FromCache bool

	// ArtifactsRestored is the number of artifacts (for cached results).
	ArtifactsRestored int
}

// Run executes a task or replays from cache.
//
// The execution flow:
//  1. Validate task
//  2. Resolve inputs
//  3. Compute hash
//  4. Check cache → if hit, replay and return
//  5. Execute task
//  6. If success (exit code 0): harvest artifacts, cache, return
//  7. If failure (non-zero): cache stdout/stderr/exitcode (NO artifacts), return
//
// From spec.md Failure Behavior:
//
//	"Failed tasks MUST NOT partially update artifacts."
//
// This means on failure, we do NOT harvest artifacts - they may be incomplete.
// We cache the failure so it can be deterministically replayed.
func (r *Runner) Run(ctx context.Context, task *Task) (*RunResult, error) {
	// Validate task
	if err := r.validateTask(task); err != nil {
		return nil, err
	}

	// Resolve inputs
	inputSet, err := r.Resolver.Resolve(task.Inputs)
	if err != nil {
		return nil, fmt.Errorf("resolving inputs: %w", err)
	}

	// Compute hash
	hashInput := HashInput{
		Inputs:     inputSet,
		Command:    task.Run,
		Env:        task.Env,
		Outputs:    task.Outputs,
		WorkingDir: r.WorkingDir,
	}
	hash := r.Hasher.ComputeHash(hashInput)

	// Check cache
	exists, err := r.Cache.Has(hash)
	if err != nil {
		return nil, fmt.Errorf("checking cache: %w", err)
	}

	if exists {
		// Cache hit - replay
		return r.replayFromCache(hash)
	}

	// Cache miss - execute
	return r.executeAndCache(ctx, task, hash)
}

// validateTask ensures the task is valid before execution.
func (r *Runner) validateTask(task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.Run == "" {
		return fmt.Errorf("task run command is required")
	}
	return nil
}

// replayFromCache retrieves and replays a cached result.
func (r *Runner) replayFromCache(hash TaskHash) (*RunResult, error) {
	entry, err := r.Cache.Get(hash)
	if err != nil {
		return nil, fmt.Errorf("retrieving cache entry: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("cache entry disappeared")
	}

	replayResult, err := r.Replayer.Replay(entry)
	if err != nil {
		return nil, fmt.Errorf("replaying cached result: %w", err)
	}

	return &RunResult{
		Hash:              hash,
		Stdout:            replayResult.Stdout,
		Stderr:            replayResult.Stderr,
		ExitCode:          replayResult.ExitCode,
		FromCache:         true,
		ArtifactsRestored: replayResult.ArtifactsRestored,
	}, nil
}

// executeAndCache runs the task and caches the result.
//
// CRITICAL: Failed tasks (non-zero exit) are cached WITHOUT artifacts.
// This ensures "Failed tasks MUST NOT partially update artifacts."
func (r *Runner) executeAndCache(ctx context.Context, task *Task, hash TaskHash) (*RunResult, error) {
	// Execute task
	execResult, err := r.Executor.Execute(ctx, task, hash)
	if err != nil {
		return nil, fmt.Errorf("executing task: %w", err)
	}

	// Prepare cache entry
	entry := &CacheEntry{
		Hash:     hash,
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
		ExitCode: execResult.ExitCode,
	}

	// Handle artifacts based on exit code
	if execResult.ExitCode == 0 {
		// SUCCESS: Harvest artifacts
		artifacts, err := r.harvestArtifacts(task.Outputs)
		if err != nil {
			return nil, fmt.Errorf("harvesting artifacts: %w", err)
		}
		entry.Artifacts = artifacts
	} else {
		// FAILURE: Do NOT harvest artifacts
		// From spec.md: "Failed tasks MUST NOT partially update artifacts."
		// We cache the failure WITHOUT artifacts
		entry.Artifacts = []CachedArtifact{}
	}

	// Store in cache
	if err := r.Cache.Put(entry); err != nil {
		return nil, fmt.Errorf("caching result: %w", err)
	}

	return &RunResult{
		Hash:              hash,
		Stdout:            execResult.Stdout,
		Stderr:            execResult.Stderr,
		ExitCode:          execResult.ExitCode,
		FromCache:         false,
		ArtifactsRestored: 0,
	}, nil
}

// harvestArtifacts collects artifacts from declared outputs.
func (r *Runner) harvestArtifacts(outputs []string) ([]CachedArtifact, error) {
	if len(outputs) == 0 {
		return []CachedArtifact{}, nil
	}

	artifactSet, err := r.Harvester.Harvest(outputs)
	if err != nil {
		return nil, err
	}

	cached := make([]CachedArtifact, len(artifactSet.Artifacts))
	for i, a := range artifactSet.Artifacts {
		cached[i] = CachedArtifact{
			Path:    a.Path,
			Content: a.Content,
		}
	}

	return cached, nil
}

// CleanArtifacts removes existing artifacts before execution.
// This ensures failed tasks don't leave stale artifacts.
//
// Call this before Run() if you want to ensure a clean workspace.
func (r *Runner) CleanArtifacts(outputs []string) error {
	for _, output := range outputs {
		fullPath := output
		if !filepath.IsAbs(output) {
			fullPath = filepath.Join(r.WorkingDir, output)
		}

		// Remove if exists (ignore if doesn't exist)
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("removing %q: %w", output, err)
		}
	}
	return nil
}
