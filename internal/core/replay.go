// Package core defines the domain models for deterministic task execution.
package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReplayResult contains the results of replaying a cached execution.
type ReplayResult struct {
	// Stdout is the cached standard output.
	Stdout []byte

	// Stderr is the cached standard error.
	Stderr []byte

	// ExitCode is the cached exit code.
	ExitCode int

	// Hash is the TaskHash that was replayed.
	Hash TaskHash

	// ArtifactsRestored is the number of artifacts restored to the workspace.
	ArtifactsRestored int
}

// Replayer restores cached execution results to the workspace.
//
// From spec.md Cache Behavior:
//   - Cached results are replayed exactly.
//
// From data-dictionary.md Replay:
//   - Includes: Bit-for-bit identical outputs
//   - Excludes: Any execution side effects
//
// From tdd.md Test 7:
//   - stdout, stderr, exit code, and artifacts MUST match exactly on replay.
type Replayer struct {
	// WorkingDir is the directory where artifacts are restored.
	WorkingDir string
}

// NewReplayer creates a new Replayer with the given working directory.
func NewReplayer(workingDir string) *Replayer {
	return &Replayer{WorkingDir: workingDir}
}

// Replay restores a cached execution result to the workspace.
//
// The replay process:
//  1. Restore each artifact to its original path (relative to WorkingDir)
//  2. Return stdout, stderr, and exit code exactly as cached
//
// Artifacts are restored with their exact cached content (bit-for-bit identical).
// Parent directories are created as needed.
//
// Returns an error if:
//   - An artifact cannot be written
//   - The cache entry is nil
func (r *Replayer) Replay(entry *CacheEntry) (*ReplayResult, error) {
	if entry == nil {
		return nil, fmt.Errorf("cache entry is nil")
	}

	// Restore artifacts to workspace
	for _, artifact := range entry.Artifacts {
		if err := r.restoreArtifact(artifact); err != nil {
			return nil, fmt.Errorf("restoring artifact %q: %w", artifact.Path, err)
		}
	}

	return &ReplayResult{
		Stdout:            entry.Stdout,
		Stderr:            entry.Stderr,
		ExitCode:          entry.ExitCode,
		Hash:              entry.Hash,
		ArtifactsRestored: len(entry.Artifacts),
	}, nil
}

// restoreArtifact writes a cached artifact to the workspace.
func (r *Replayer) restoreArtifact(artifact CachedArtifact) error {
	// Determine target path
	targetPath := artifact.Path
	if !filepath.IsAbs(artifact.Path) {
		targetPath = filepath.Join(r.WorkingDir, artifact.Path)
	}

	// Convert forward slashes to OS path separator
	targetPath = filepath.FromSlash(targetPath)

	// Create parent directories
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Write artifact content (bit-for-bit identical)
	if err := os.WriteFile(targetPath, artifact.Content, 0644); err != nil {
		return fmt.Errorf("writing artifact: %w", err)
	}

	return nil
}
