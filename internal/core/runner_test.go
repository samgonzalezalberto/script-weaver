package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRunner_FailedTasksCacheable verifies spec.md:
// "Failed executions (non-zero exit code) are cacheable."
func TestRunner_FailedTasksCacheable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	task := &Task{
		Name:   "failing-task",
		Inputs: []string{},
		Run:    "echo 'error message' >&2; exit 1",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First run - should execute and cache failure
	result1, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result1.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result1.ExitCode)
	}
	if result1.FromCache {
		t.Error("first run should not be from cache")
	}

	// Second run - should be from cache
	result2, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !result2.FromCache {
		t.Error("second run should be from cache")
	}
	if result2.ExitCode != 1 {
		t.Errorf("cached exit code wrong: %d", result2.ExitCode)
	}
}

// TestRunner_FailedTaskReplayIdentical verifies spec.md:
// "Replaying a failed task MUST return:
//   - Identical stdout
//   - Identical stderr
//   - Identical exit code"
func TestRunner_FailedTaskReplayIdentical(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	task := &Task{
		Name:   "failing-task",
		Inputs: []string{},
		Run:    "echo 'stdout message'; echo 'stderr message' >&2; exit 42",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First run
	result1, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Second run (from cache)
	result2, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify bit-for-bit identical
	if !bytes.Equal(result1.Stdout, result2.Stdout) {
		t.Errorf("stdout mismatch:\noriginal: %q\nreplayed: %q", result1.Stdout, result2.Stdout)
	}
	if !bytes.Equal(result1.Stderr, result2.Stderr) {
		t.Errorf("stderr mismatch:\noriginal: %q\nreplayed: %q", result1.Stderr, result2.Stderr)
	}
	if result1.ExitCode != result2.ExitCode {
		t.Errorf("exit code mismatch: %d != %d", result1.ExitCode, result2.ExitCode)
	}
}

// TestRunner_FailedTaskNoPartialArtifacts verifies spec.md:
// "Failed tasks MUST NOT partially update artifacts."
func TestRunner_FailedTaskNoPartialArtifacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	// Task that creates a file but then fails
	task := &Task{
		Name:    "partial-fail",
		Inputs:  []string{},
		Run:     "echo 'partial content' > output.txt; exit 1",
		Outputs: []string{"output.txt"},
		Env:     map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}

	// Verify that no artifacts were cached for the failed task
	entry, err := cache.Get(result.Hash)
	if err != nil {
		t.Fatalf("Get cache failed: %v", err)
	}

	if len(entry.Artifacts) != 0 {
		t.Errorf("failed task should have 0 cached artifacts, got %d", len(entry.Artifacts))
	}
}

// TestRunner_SuccessfulTaskHasArtifacts verifies successful tasks cache artifacts.
func TestRunner_SuccessfulTaskHasArtifacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	task := &Task{
		Name:    "success-task",
		Inputs:  []string{},
		Run:     "echo 'content' > output.txt",
		Outputs: []string{"output.txt"},
		Env:     map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Verify artifacts were cached
	entry, err := cache.Get(result.Hash)
	if err != nil {
		t.Fatalf("Get cache failed: %v", err)
	}

	if len(entry.Artifacts) != 1 {
		t.Errorf("successful task should have 1 artifact, got %d", len(entry.Artifacts))
	}
}

// TestRunner_CacheHitSkipsExecution verifies spec.md:
// "If a Task Hash has been seen before, the task MUST NOT be re-executed."
func TestRunner_CacheHitSkipsExecution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	// Create a marker file to detect execution
	markerFile := filepath.Join(tmpDir, "marker.txt")

	task := &Task{
		Name:   "marker-task",
		Inputs: []string{},
		// Append to marker file each time
		Run: fmt.Sprintf("echo 'executed' >> %s", markerFile),
		Env: map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First run - executes
	result1, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result1.FromCache {
		t.Error("first run should not be from cache")
	}

	// Read marker after first run
	content1, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("failed to read marker: %v", err)
	}

	// Second run - should be from cache, NOT execute
	result2, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !result2.FromCache {
		t.Error("second run should be from cache")
	}

	// Read marker after second run - should be unchanged
	content2, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("failed to read marker: %v", err)
	}

	if !bytes.Equal(content1, content2) {
		t.Errorf("task was re-executed! marker changed:\nbefore: %q\nafter: %q", content1, content2)
	}
}

// TestRunner_FailureIsDeterministic verifies that failures are reproducible.
// A failed task with the same inputs should always produce the same failure.
func TestRunner_FailureIsDeterministic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	task := &Task{
		Name:   "deterministic-fail",
		Inputs: []string{},
		Run:    "echo 'deterministic error'; exit 5",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run multiple times
	var results []*RunResult
	for i := 0; i < 3; i++ {
		result, err := runner.Run(ctx, task)
		if err != nil {
			t.Fatalf("Run %d failed: %v", i, err)
		}
		results = append(results, result)
	}

	// All should have same hash, stdout, stderr, exit code
	for i := 1; i < len(results); i++ {
		if results[i].Hash != results[0].Hash {
			t.Errorf("run %d has different hash", i)
		}
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Errorf("run %d has different stdout", i)
		}
		if !bytes.Equal(results[i].Stderr, results[0].Stderr) {
			t.Errorf("run %d has different stderr", i)
		}
		if results[i].ExitCode != results[0].ExitCode {
			t.Errorf("run %d has different exit code", i)
		}
	}
}

// TestRunner_ValidatesTask verifies task validation.
func TestRunner_ValidatesTask(t *testing.T) {
	cache := NewMemoryCache()
	runner := NewRunner("/tmp", cache)

	ctx := context.Background()

	// Nil task
	_, err := runner.Run(ctx, nil)
	if err == nil {
		t.Error("expected error for nil task")
	}

	// Empty name
	_, err = runner.Run(ctx, &Task{Name: "", Run: "echo"})
	if err == nil {
		t.Error("expected error for empty name")
	}

	// Empty run
	_, err = runner.Run(ctx, &Task{Name: "test", Run: ""})
	if err == nil {
		t.Error("expected error for empty run")
	}
}

// TestRunner_CleanArtifacts verifies artifact cleanup.
func TestRunner_CleanArtifacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create existing artifacts
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("old"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	existingDir := filepath.Join(tmpDir, "existing-dir")
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	nestedFile := filepath.Join(existingDir, "nested.txt")
	if err := os.WriteFile(nestedFile, []byte("nested"), 0644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	// Clean artifacts
	err = runner.CleanArtifacts([]string{"existing.txt", "existing-dir"})
	if err != nil {
		t.Fatalf("CleanArtifacts failed: %v", err)
	}

	// Verify removed
	if _, err := os.Stat(existingFile); !os.IsNotExist(err) {
		t.Error("existing.txt should be removed")
	}
	if _, err := os.Stat(existingDir); !os.IsNotExist(err) {
		t.Error("existing-dir should be removed")
	}
}

// TestRunner_ReplayRestoresArtifacts verifies artifacts are restored on replay.
func TestRunner_ReplayRestoresArtifacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	task := &Task{
		Name:    "artifact-task",
		Inputs:  []string{},
		Run:     "echo 'artifact content' > output.txt",
		Outputs: []string{"output.txt"},
		Env:     map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First run - creates artifact
	result1, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Read original artifact content
	originalContent, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		t.Fatalf("failed to read artifact: %v", err)
	}

	// Delete the artifact
	os.Remove(filepath.Join(tmpDir, "output.txt"))

	// Second run - should restore from cache
	result2, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !result2.FromCache {
		t.Error("second run should be from cache")
	}

	// Artifact should be restored
	restoredContent, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		t.Fatalf("failed to read restored artifact: %v", err)
	}

	if !bytes.Equal(originalContent, restoredContent) {
		t.Errorf("restored artifact mismatch:\noriginal: %q\nrestored: %q", originalContent, restoredContent)
	}

	// Verify same hash
	if result1.Hash != result2.Hash {
		t.Errorf("hash mismatch: %s != %s", result1.Hash, result2.Hash)
	}
}
