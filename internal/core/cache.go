// Package core defines the domain models for deterministic task execution.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CacheEntry represents a stored result of a task execution.
//
// From data-dictionary.md:
//
//	Includes: stdout, stderr, exit code, artifacts
//	Excludes: Execution timestamps, Host-specific metadata
//
// From spec.md Cache Behavior:
//
//	Failed executions (non-zero exit code) are cacheable.
type CacheEntry struct {
	// Hash is the TaskHash that identifies this cache entry.
	Hash TaskHash `json:"hash"`

	// Stdout is the captured standard output.
	Stdout []byte `json:"stdout"`

	// Stderr is the captured standard error.
	Stderr []byte `json:"stderr"`

	// ExitCode is the process exit code.
	ExitCode int `json:"exit_code"`

	// Artifacts contains the harvested output files.
	Artifacts []CachedArtifact `json:"artifacts"`
}

// CachedArtifact represents a single artifact stored in the cache.
type CachedArtifact struct {
	// Path is the normalized path of the artifact.
	Path string `json:"path"`

	// Content is the artifact file content.
	Content []byte `json:"content"`
}

// Cache provides storage and retrieval of task execution results.
//
// From spec.md Cache Behavior:
//   - If a Task Hash has been seen before, the task MUST NOT be re-executed.
//   - Cached results are replayed exactly.
//
// From tdd.md Test 2:
//   - Same Hash Prevents Re-execution
//
// From tdd.md Test 7:
//   - Cache Replay Is Bit-for-Bit Identical
type Cache interface {
	// Has checks if a cache entry exists for the given hash.
	Has(hash TaskHash) (bool, error)

	// Get retrieves a cache entry by hash.
	// Returns nil if the entry does not exist.
	Get(hash TaskHash) (*CacheEntry, error)

	// Put stores a cache entry.
	Put(entry *CacheEntry) error
}

// FileCache implements Cache using the filesystem.
//
// Structure:
//
//	{CacheDir}/
//	  {hash[0:2]}/
//	    {hash}/
//	      metadata.json  (stdout, stderr, exit_code, artifact paths)
//	      artifacts/
//	        {artifact-hash}.blob
type FileCache struct {
	// CacheDir is the root directory for cache storage.
	CacheDir string
}

// NewFileCache creates a new filesystem-based cache.
func NewFileCache(cacheDir string) *FileCache {
	return &FileCache{CacheDir: cacheDir}
}

// Has checks if a cache entry exists for the given hash.
func (c *FileCache) Has(hash TaskHash) (bool, error) {
	entryDir := c.entryPath(hash)
	metadataPath := filepath.Join(entryDir, "metadata.json")

	_, err := os.Stat(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking cache entry: %w", err)
	}

	return true, nil
}

// Get retrieves a cache entry by hash.
func (c *FileCache) Get(hash TaskHash) (*CacheEntry, error) {
	entryDir := c.entryPath(hash)
	metadataPath := filepath.Join(entryDir, "metadata.json")

	// Read metadata
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache metadata: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing cache metadata: %w", err)
	}

	// Read artifact contents
	artifactsDir := filepath.Join(entryDir, "artifacts")
	for i := range entry.Artifacts {
		blobPath := filepath.Join(artifactsDir, fmt.Sprintf("%d.blob", i))
		content, err := os.ReadFile(blobPath)
		if err != nil {
			return nil, fmt.Errorf("reading artifact %d: %w", i, err)
		}
		entry.Artifacts[i].Content = content
	}

	return &entry, nil
}

// Put stores a cache entry.
func (c *FileCache) Put(entry *CacheEntry) error {
	if entry == nil {
		return fmt.Errorf("cache entry is nil")
	}

	entryDir := c.entryPath(entry.Hash)
	parentDir := filepath.Dir(entryDir)

	// Ensure parent exists so temp dir is created on the same filesystem.
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	// Write into a temp entry dir, then rename into place.
	// This prevents crashes from leaving corrupt metadata.json (or partial blobs)
	// at the canonical entry path.
	tmpDir, err := os.MkdirTemp(parentDir, "tmp-entry-"+string(entry.Hash)+"-")
	if err != nil {
		return fmt.Errorf("creating temp cache entry dir: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = os.RemoveAll(tmpDir)
	}()

	artifactsDir := filepath.Join(tmpDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("creating cache artifacts dir: %w", err)
	}

	// Write artifact blobs first (so metadata only appears after blobs succeed).
	for i, artifact := range entry.Artifacts {
		blobPath := filepath.Join(artifactsDir, fmt.Sprintf("%d.blob", i))
		if err := writeFileAtomic(blobPath, artifact.Content, 0644); err != nil {
			return fmt.Errorf("writing artifact %d: %w", i, err)
		}
	}

	// Create metadata (without content to save space - content is in blobs)
	metadata := CacheEntry{
		Hash:     entry.Hash,
		Stdout:   entry.Stdout,
		Stderr:   entry.Stderr,
		ExitCode: entry.ExitCode,
		Artifacts: make([]CachedArtifact, len(entry.Artifacts)),
	}
	for i, a := range entry.Artifacts {
		metadata.Artifacts[i] = CachedArtifact{
			Path:    a.Path,
			Content: nil, // Content stored in blob files
		}
	}

	// Write metadata
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache metadata: %w", err)
	}

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	if err := writeFileAtomic(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("writing cache metadata: %w", err)
	}

	// Best-effort remove of any existing entry; a crash between remove and rename
	// yields a cache miss (safe), not corruption.
	_ = os.RemoveAll(entryDir)
	if err := os.Rename(tmpDir, entryDir); err != nil {
		return fmt.Errorf("committing cache entry: %w", err)
	}
	committed = true
	return nil
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

// entryPath returns the directory path for a cache entry.
// Uses first 2 characters of hash as a prefix directory to avoid
// having too many entries in a single directory.
func (c *FileCache) entryPath(hash TaskHash) string {
	hashStr := string(hash)
	if len(hashStr) < 2 {
		return filepath.Join(c.CacheDir, hashStr)
	}
	return filepath.Join(c.CacheDir, hashStr[:2], hashStr)
}

// MemoryCache implements Cache using in-memory storage.
// Useful for testing and short-lived processes.
type MemoryCache struct {
	entries map[TaskHash]*CacheEntry
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries: make(map[TaskHash]*CacheEntry),
	}
}

// Has checks if a cache entry exists.
func (c *MemoryCache) Has(hash TaskHash) (bool, error) {
	_, exists := c.entries[hash]
	return exists, nil
}

// Get retrieves a cache entry.
func (c *MemoryCache) Get(hash TaskHash) (*CacheEntry, error) {
	entry, exists := c.entries[hash]
	if !exists {
		return nil, nil
	}
	// Return a copy to prevent mutation
	return c.copyEntry(entry), nil
}

// Put stores a cache entry.
func (c *MemoryCache) Put(entry *CacheEntry) error {
	if entry == nil {
		return fmt.Errorf("cache entry is nil")
	}
	// Store a copy to prevent mutation
	c.entries[entry.Hash] = c.copyEntry(entry)
	return nil
}

// copyEntry creates a deep copy of a cache entry.
func (c *MemoryCache) copyEntry(entry *CacheEntry) *CacheEntry {
	copy := &CacheEntry{
		Hash:      entry.Hash,
		Stdout:    make([]byte, len(entry.Stdout)),
		Stderr:    make([]byte, len(entry.Stderr)),
		ExitCode:  entry.ExitCode,
		Artifacts: make([]CachedArtifact, len(entry.Artifacts)),
	}
	
	// Use the built-in copy function for byte slices
	builtinCopy(copy.Stdout, entry.Stdout)
	builtinCopy(copy.Stderr, entry.Stderr)
	
	for i, a := range entry.Artifacts {
		copy.Artifacts[i] = CachedArtifact{
			Path:    a.Path,
			Content: make([]byte, len(a.Content)),
		}
		builtinCopy(copy.Artifacts[i].Content, a.Content)
	}
	
	return copy
}

// builtinCopy wraps the built-in copy function.
func builtinCopy(dst, src []byte) {
	copy(dst, src)
}
