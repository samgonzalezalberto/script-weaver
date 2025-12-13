package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCache_SameHashPreventsReExecution verifies tdd.md#Test-2:
// "Given a previously executed task, an identical Task Hash:
// The task MUST NOT execute again. Cached results MUST be replayed."
func TestCache_SameHashPreventsReExecution(t *testing.T) {
	cache := NewMemoryCache()

	hash := TaskHash("abc123def456")

	// First check - should not exist
	exists, err := cache.Has(hash)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if exists {
		t.Error("hash should not exist initially")
	}

	// Store result
	entry := &CacheEntry{
		Hash:     hash,
		Stdout:   []byte("output"),
		Stderr:   []byte("error"),
		ExitCode: 0,
	}
	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Second check - should exist (prevents re-execution)
	exists, err = cache.Has(hash)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !exists {
		t.Error("hash should exist after Put")
	}
}

// TestCache_ReplayBitForBitIdentical verifies tdd.md#Test-7:
// "stdout, stderr, exit code, and artifacts MUST match exactly on replay."
func TestCache_ReplayBitForBitIdentical(t *testing.T) {
	cache := NewMemoryCache()

	// Original execution result
	original := &CacheEntry{
		Hash:     TaskHash("test-hash"),
		Stdout:   []byte("exact stdout content\nwith newlines\n"),
		Stderr:   []byte("exact stderr content\n"),
		ExitCode: 42,
		Artifacts: []CachedArtifact{
			{Path: "output/file1.txt", Content: []byte("file1 content")},
			{Path: "output/file2.bin", Content: []byte{0x00, 0x01, 0x02, 0xff}},
		},
	}

	// Store
	if err := cache.Put(original); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Retrieve
	retrieved, err := cache.Get(original.Hash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify bit-for-bit identical
	if !bytes.Equal(retrieved.Stdout, original.Stdout) {
		t.Errorf("stdout mismatch:\noriginal: %q\nretrieved: %q", original.Stdout, retrieved.Stdout)
	}

	if !bytes.Equal(retrieved.Stderr, original.Stderr) {
		t.Errorf("stderr mismatch:\noriginal: %q\nretrieved: %q", original.Stderr, retrieved.Stderr)
	}

	if retrieved.ExitCode != original.ExitCode {
		t.Errorf("exit code mismatch: %d != %d", retrieved.ExitCode, original.ExitCode)
	}

	if len(retrieved.Artifacts) != len(original.Artifacts) {
		t.Fatalf("artifact count mismatch: %d != %d", len(retrieved.Artifacts), len(original.Artifacts))
	}

	for i := range original.Artifacts {
		if retrieved.Artifacts[i].Path != original.Artifacts[i].Path {
			t.Errorf("artifact %d path mismatch", i)
		}
		if !bytes.Equal(retrieved.Artifacts[i].Content, original.Artifacts[i].Content) {
			t.Errorf("artifact %d content mismatch", i)
		}
	}
}

// TestCache_FailedExecutionsCacheable verifies spec.md:
// "Failed executions (non-zero exit code) are cacheable."
func TestCache_FailedExecutionsCacheable(t *testing.T) {
	cache := NewMemoryCache()

	// Failed execution
	entry := &CacheEntry{
		Hash:     TaskHash("failed-task"),
		Stdout:   []byte("partial output"),
		Stderr:   []byte("error: something failed\n"),
		ExitCode: 1,
	}

	// Should store successfully
	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Should retrieve successfully
	retrieved, err := cache.Get(entry.Hash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ExitCode != 1 {
		t.Errorf("failed exit code not preserved: %d", retrieved.ExitCode)
	}
}

// TestCache_GetNonExistent returns nil.
func TestCache_GetNonExistent(t *testing.T) {
	cache := NewMemoryCache()

	result, err := cache.Get(TaskHash("does-not-exist"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil for non-existent entry")
	}
}

// TestCache_PutNilFails returns error.
func TestCache_PutNilFails(t *testing.T) {
	cache := NewMemoryCache()

	err := cache.Put(nil)
	if err == nil {
		t.Error("expected error for nil entry")
	}
}

// TestMemoryCache_IsolatesMutations verifies stored entries are copied.
func TestMemoryCache_IsolatesMutations(t *testing.T) {
	cache := NewMemoryCache()

	entry := &CacheEntry{
		Hash:   TaskHash("test"),
		Stdout: []byte("original"),
	}

	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Mutate the original
	entry.Stdout[0] = 'X'

	// Retrieved should be unchanged
	retrieved, _ := cache.Get(entry.Hash)
	if retrieved.Stdout[0] == 'X' {
		t.Error("cache did not isolate mutations")
	}
}

// TestFileCache_PersistsToFilesystem verifies filesystem storage.
func TestFileCache_PersistsToFilesystem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewFileCache(tmpDir)

	entry := &CacheEntry{
		Hash:     TaskHash("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		Stdout:   []byte("stdout content"),
		Stderr:   []byte("stderr content"),
		ExitCode: 0,
		Artifacts: []CachedArtifact{
			{Path: "output.txt", Content: []byte("artifact content")},
		},
	}

	// Store
	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify directory structure
	entryDir := filepath.Join(tmpDir, "ab", string(entry.Hash))
	if _, err := os.Stat(entryDir); os.IsNotExist(err) {
		t.Error("entry directory not created")
	}

	metadataPath := filepath.Join(entryDir, "metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("metadata.json not created")
	}

	artifactPath := filepath.Join(entryDir, "artifacts", "0.blob")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Error("artifact blob not created")
	}

	// Retrieve and verify
	retrieved, err := cache.Get(entry.Hash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !bytes.Equal(retrieved.Stdout, entry.Stdout) {
		t.Error("stdout mismatch")
	}
	if !bytes.Equal(retrieved.Artifacts[0].Content, entry.Artifacts[0].Content) {
		t.Error("artifact content mismatch")
	}
}

// TestFileCache_HasWorks verifies Has operation.
func TestFileCache_HasWorks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewFileCache(tmpDir)
	hash := TaskHash("testhash123")

	// Initially should not exist
	exists, err := cache.Has(hash)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if exists {
		t.Error("should not exist initially")
	}

	// Store entry
	entry := &CacheEntry{Hash: hash, Stdout: []byte("test")}
	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Now should exist
	exists, err = cache.Has(hash)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !exists {
		t.Error("should exist after Put")
	}
}

// TestFileCache_GetNonExistent returns nil.
func TestFileCache_GetNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewFileCache(tmpDir)

	result, err := cache.Get(TaskHash("does-not-exist"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil for non-existent entry")
	}
}

// TestCache_NoTimestampsStored verifies data-dictionary.md:
// "Excludes: Execution timestamps, Host-specific metadata"
func TestCache_NoTimestampsStored(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewFileCache(tmpDir)

	entry := &CacheEntry{
		Hash:     TaskHash("test"),
		Stdout:   []byte("output"),
		ExitCode: 0,
	}

	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Read the metadata file directly
	metadataPath := filepath.Join(tmpDir, "te", "test", "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}

	// Verify no timestamp fields
	content := string(data)
	if bytes.Contains(data, []byte("timestamp")) ||
		bytes.Contains(data, []byte("created")) ||
		bytes.Contains(data, []byte("modified")) ||
		bytes.Contains(data, []byte("time")) {
		t.Errorf("metadata contains timestamp-like fields: %s", content)
	}
}
