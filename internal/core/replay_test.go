package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestReplay_RestoresArtifacts verifies artifacts are restored to workspace.
func TestReplay_RestoresArtifacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	entry := &CacheEntry{
		Hash:     TaskHash("test-hash"),
		Stdout:   []byte("stdout"),
		Stderr:   []byte("stderr"),
		ExitCode: 0,
		Artifacts: []CachedArtifact{
			{Path: "output.txt", Content: []byte("artifact content")},
			{Path: "subdir/nested.txt", Content: []byte("nested content")},
		},
	}

	result, err := replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// Verify artifacts were restored
	if result.ArtifactsRestored != 2 {
		t.Errorf("expected 2 artifacts restored, got %d", result.ArtifactsRestored)
	}

	// Check first artifact
	content1, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		t.Fatalf("failed to read restored artifact: %v", err)
	}
	if string(content1) != "artifact content" {
		t.Errorf("artifact 1 content mismatch: %s", content1)
	}

	// Check nested artifact
	content2, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "nested.txt"))
	if err != nil {
		t.Fatalf("failed to read nested artifact: %v", err)
	}
	if string(content2) != "nested content" {
		t.Errorf("artifact 2 content mismatch: %s", content2)
	}
}

// TestReplay_BitForBitIdentical verifies tdd.md#Test-7:
// "stdout, stderr, exit code, and artifacts MUST match exactly on replay."
func TestReplay_BitForBitIdentical(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	// Include binary content to verify bit-for-bit
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe, 0xfd}

	entry := &CacheEntry{
		Hash:     TaskHash("test-hash"),
		Stdout:   []byte("exact stdout\n"),
		Stderr:   []byte("exact stderr\n"),
		ExitCode: 42,
		Artifacts: []CachedArtifact{
			{Path: "binary.bin", Content: binaryContent},
		},
	}

	result, err := replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// Verify stdout is bit-for-bit
	if !bytes.Equal(result.Stdout, entry.Stdout) {
		t.Errorf("stdout not bit-for-bit identical")
	}

	// Verify stderr is bit-for-bit
	if !bytes.Equal(result.Stderr, entry.Stderr) {
		t.Errorf("stderr not bit-for-bit identical")
	}

	// Verify exit code
	if result.ExitCode != entry.ExitCode {
		t.Errorf("exit code mismatch: %d != %d", result.ExitCode, entry.ExitCode)
	}

	// Verify binary artifact is bit-for-bit
	content, err := os.ReadFile(filepath.Join(tmpDir, "binary.bin"))
	if err != nil {
		t.Fatalf("failed to read artifact: %v", err)
	}
	if !bytes.Equal(content, binaryContent) {
		t.Errorf("binary artifact not bit-for-bit identical")
	}
}

// TestReplay_PreservesExitCode verifies exit code is preserved.
func TestReplay_PreservesExitCode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	testCases := []int{0, 1, 42, 127, 255}

	for _, exitCode := range testCases {
		entry := &CacheEntry{
			Hash:     TaskHash("test"),
			ExitCode: exitCode,
		}

		result, err := replayer.Replay(entry)
		if err != nil {
			t.Fatalf("Replay failed for exit code %d: %v", exitCode, err)
		}

		if result.ExitCode != exitCode {
			t.Errorf("exit code %d not preserved, got %d", exitCode, result.ExitCode)
		}
	}
}

// TestReplay_FailedTaskReplay verifies spec.md:
// "Replaying a failed task MUST return identical stdout, stderr, exit code."
func TestReplay_FailedTaskReplay(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	// Failed task with non-zero exit code
	entry := &CacheEntry{
		Hash:     TaskHash("failed-task"),
		Stdout:   []byte("partial output before failure\n"),
		Stderr:   []byte("error: compilation failed\n/path/to/file.go:10: undefined: foo\n"),
		ExitCode: 1,
		Artifacts: []CachedArtifact{}, // No artifacts for failed task
	}

	result, err := replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// All must be identical
	if !bytes.Equal(result.Stdout, entry.Stdout) {
		t.Error("failed task stdout not preserved")
	}
	if !bytes.Equal(result.Stderr, entry.Stderr) {
		t.Error("failed task stderr not preserved")
	}
	if result.ExitCode != 1 {
		t.Errorf("failed task exit code not preserved: %d", result.ExitCode)
	}
}

// TestReplay_CreatesParentDirectories verifies nested paths work.
func TestReplay_CreatesParentDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	entry := &CacheEntry{
		Hash: TaskHash("test"),
		Artifacts: []CachedArtifact{
			{Path: "deep/nested/path/to/file.txt", Content: []byte("content")},
		},
	}

	_, err = replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, "deep", "nested", "path", "to", "file.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("deeply nested file not created")
	}
}

// TestReplay_NilEntryFails returns error.
func TestReplay_NilEntryFails(t *testing.T) {
	replayer := NewReplayer("/tmp")

	_, err := replayer.Replay(nil)
	if err == nil {
		t.Error("expected error for nil entry")
	}
}

// TestReplay_EmptyArtifacts succeeds.
func TestReplay_EmptyArtifacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	entry := &CacheEntry{
		Hash:      TaskHash("test"),
		Stdout:    []byte("output"),
		Artifacts: []CachedArtifact{},
	}

	result, err := replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if result.ArtifactsRestored != 0 {
		t.Errorf("expected 0 artifacts, got %d", result.ArtifactsRestored)
	}
}

// TestReplay_HashPreserved verifies hash is in result.
func TestReplay_HashPreserved(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	replayer := NewReplayer(tmpDir)

	expectedHash := TaskHash("expected-hash-value")
	entry := &CacheEntry{
		Hash: expectedHash,
	}

	result, err := replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if result.Hash != expectedHash {
		t.Errorf("hash mismatch: %s != %s", result.Hash, expectedHash)
	}
}

// TestReplay_OverwritesExistingFiles verifies existing files are replaced.
func TestReplay_OverwritesExistingFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replay-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create existing file with different content
	existingPath := filepath.Join(tmpDir, "output.txt")
	if err := os.WriteFile(existingPath, []byte("old content"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	replayer := NewReplayer(tmpDir)

	entry := &CacheEntry{
		Hash: TaskHash("test"),
		Artifacts: []CachedArtifact{
			{Path: "output.txt", Content: []byte("new cached content")},
		},
	}

	_, err = replayer.Replay(entry)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// Verify content was replaced
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "new cached content" {
		t.Errorf("file not overwritten: %s", content)
	}
}
