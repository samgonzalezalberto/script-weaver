// Package core defines the domain models for deterministic task execution.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// InputResolver resolves declared input patterns to a deterministic InputSet.
//
// From spec.md Deterministic Guarantees - Input Determinism:
//   - Input files are read by content.
//   - Glob expansion is strictly sorted.
//   - File ordering is stable across runs and machines.
//
// From tdd.md Test 9:
//   - Expanded file list MUST be strictly sorted.
//   - Different filesystem ordering MUST NOT affect hashing or execution.
type InputResolver struct {
	// BaseDir is the working directory for resolving relative paths.
	// All paths are resolved relative to this directory.
	BaseDir string
}

// NewInputResolver creates a new InputResolver with the given base directory.
func NewInputResolver(baseDir string) *InputResolver {
	return &InputResolver{BaseDir: baseDir}
}

// Resolve expands all input patterns and returns a deterministic InputSet.
//
// The resolution process:
//  1. Each pattern is expanded using filepath.Glob
//  2. All expanded paths are collected
//  3. Paths are normalized to use forward slashes
//  4. Paths are strictly sorted lexicographically
//  5. Duplicates are removed
//  6. File contents are read (content-based identity, not metadata)
//
// Returns an error if:
//   - A pattern is invalid
//   - A file cannot be read
//   - No files match any pattern (optional: configurable behavior)
func (r *InputResolver) Resolve(patterns []string) (*InputSet, error) {
	if len(patterns) == 0 {
		return &InputSet{Inputs: []Input{}}, nil
	}

	// Collect all expanded paths
	pathSet := make(map[string]struct{})

	for _, pattern := range patterns {
		expanded, err := r.expandPattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("expanding pattern %q: %w", pattern, err)
		}
		for _, p := range expanded {
			pathSet[p] = struct{}{}
		}
	}

	// Extract and sort paths deterministically
	// CRITICAL: Must sort explicitly, do not rely on OS directory ordering
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	// Read file contents (content-based identity)
	inputs := make([]Input, 0, len(paths))
	for _, path := range paths {
		content, err := r.readFileContent(path)
		if err != nil {
			return nil, fmt.Errorf("reading input %q: %w", path, err)
		}
		inputs = append(inputs, Input{
			Path:    path,
			Content: content,
		})
	}

	return &InputSet{Inputs: inputs}, nil
}

// expandPattern expands a single glob pattern into a sorted list of file paths.
// If the pattern contains no glob characters, it is treated as a literal path.
func (r *InputResolver) expandPattern(pattern string) ([]string, error) {
	// Resolve relative to base directory
	fullPattern := pattern
	if !filepath.IsAbs(pattern) {
		fullPattern = filepath.Join(r.BaseDir, pattern)
	}

	// Expand glob pattern
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// If no glob characters and file exists, treat as literal path
	if len(matches) == 0 && !containsGlobChar(pattern) {
		// Check if file exists
		if _, err := os.Stat(fullPattern); err == nil {
			matches = []string{fullPattern}
		}
	}

	// Normalize all paths
	normalized := make([]string, 0, len(matches))
	for _, match := range matches {
		// Skip directories - we only want files
		info, err := os.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", match, err)
		}
		if info.IsDir() {
			continue
		}

		// Normalize path separators for cross-platform determinism
		normPath := filepath.ToSlash(match)
		normalized = append(normalized, normPath)
	}

	return normalized, nil
}

// readFileContent reads the content of a file.
// Only content is read; metadata (mtime, permissions) is ignored for determinism.
func (r *InputResolver) readFileContent(path string) ([]byte, error) {
	// Convert back to OS path for reading
	osPath := filepath.FromSlash(path)
	content, err := os.ReadFile(osPath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// containsGlobChar returns true if the pattern contains glob special characters.
func containsGlobChar(pattern string) bool {
	for _, c := range pattern {
		switch c {
		case '*', '?', '[', ']':
			return true
		}
	}
	return false
}
