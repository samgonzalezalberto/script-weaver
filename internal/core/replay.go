// Package core defines the domain models for deterministic task execution.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

	restored, err := r.RestoreArtifacts(entry.Hash.String(), entry)
	if err != nil {
		return nil, err
	}

	return &ReplayResult{
		Stdout:            entry.Stdout,
		Stderr:            entry.Stderr,
		ExitCode:          entry.ExitCode,
		Hash:              entry.Hash,
		ArtifactsRestored: restored,
	}, nil
}

// RestoreArtifacts ensures the workspace artifacts for a cached task are present and correct.
//
// Sprint-02 requirement:
//   - Check if expected output files exist with correct content hashes.
//   - If missing or mismatched, restore from cache using an atomic write/replace.
//   - Fail hard if an artifact cannot be retrieved from cache.
//
// taskID is used only for error messages.
func (r *Replayer) RestoreArtifacts(taskID string, entry *CacheEntry) (int, error) {
	if r == nil {
		return 0, fmt.Errorf("replayer is nil")
	}
	if entry == nil {
		return 0, fmt.Errorf("cache entry is nil")
	}

	restored := 0
	for _, artifact := range entry.Artifacts {
		if artifact.Path == "" {
			return restored, fmt.Errorf("task %q: artifact path is empty", taskID)
		}
		if artifact.Content == nil {
			return restored, fmt.Errorf("task %q: artifact %q missing content in cache entry", taskID, artifact.Path)
		}

		targetPath, err := r.targetPathForArtifact(artifact.Path)
		if err != nil {
			return restored, fmt.Errorf("task %q: resolving artifact %q target path: %w", taskID, artifact.Path, err)
		}

		wantHash := sha256Hex(artifact.Content)
		haveHash, ok, err := fileSHA256HexIfExists(targetPath)
		if err != nil {
			return restored, fmt.Errorf("task %q: hashing existing artifact %q: %w", taskID, artifact.Path, err)
		}
		if ok && haveHash == wantHash {
			continue
		}

		if err := atomicWriteFile(targetPath, artifact.Content, 0644); err != nil {
			return restored, fmt.Errorf("task %q: restoring artifact %q: %w", taskID, artifact.Path, err)
		}
		restored++
	}

	return restored, nil
}

// restoreArtifact writes a cached artifact to the workspace.
func (r *Replayer) targetPathForArtifact(artifactPath string) (string, error) {
	// Determine target path
	targetPath := artifactPath
	if !filepath.IsAbs(artifactPath) {
		targetPath = filepath.Join(r.WorkingDir, artifactPath)
	}

	// Convert forward slashes to OS path separator
	targetPath = filepath.FromSlash(targetPath)

	// Create parent directories
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("creating parent directory: %w", err)
	}

	return targetPath, nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fileSHA256HexIfExists(path string) (hash string, exists bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", true, err
	}
	return hex.EncodeToString(h.Sum(nil)), true, nil
}

// atomicWriteFile writes content to path by writing to a temp file in the same directory
// and then renaming it over the destination.
func atomicWriteFile(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
