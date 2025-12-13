// Package core defines the domain models for deterministic task execution.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Harvester collects artifacts from declared output paths after task execution.
//
// From spec.md Output Determinism:
//   - Outputs are normalized to remove nondeterministic data (e.g., timestamps).
//   - File ordering and metadata are stable.
//
// From data-dictionary.md Artifact:
//   - Includes: Normalized file contents, Stable metadata
//   - Excludes: Undeclared files, Temporary or intermediate files
//
// From tdd.md Test 8:
//   - Only declared outputs are captured.
//   - Artifacts are stored and replayed from cache.
type Harvester struct {
	// BaseDir is the working directory where outputs are relative to.
	BaseDir string

	// Normalizer is used to normalize artifact contents.
	// If nil, no normalization is applied (raw bytes preserved).
	Normalizer OutputNormalizer
}

// OutputNormalizer defines the interface for normalizing output content.
// Normalization removes nondeterministic data like timestamps.
type OutputNormalizer interface {
	// Normalize processes content to remove nondeterministic data.
	// Returns normalized content.
	Normalize(content []byte) []byte
}

// NewHarvester creates a new Harvester with the given base directory.
func NewHarvester(baseDir string) *Harvester {
	return &Harvester{
		BaseDir:    baseDir,
		Normalizer: nil, // Default: no normalization (raw bytes)
	}
}

// NewHarvesterWithNormalizer creates a Harvester with a custom normalizer.
func NewHarvesterWithNormalizer(baseDir string, normalizer OutputNormalizer) *Harvester {
	return &Harvester{
		BaseDir:    baseDir,
		Normalizer: normalizer,
	}
}

// Harvest collects artifacts from the declared output paths.
//
// CRITICAL: Only files explicitly declared in outputs are collected.
// This does NOT scan for "all modified files" or use git status.
//
// The harvesting process:
//  1. Each declared output path is resolved relative to BaseDir
//  2. If the path is a file, it is collected
//  3. If the path is a directory, all files within are collected recursively
//  4. All collected paths are sorted for determinism
//  5. File contents are read and optionally normalized
//
// Returns an error if:
//   - A declared output does not exist (task failed to produce it)
//   - A file cannot be read
func (h *Harvester) Harvest(declaredOutputs []string) (*ArtifactSet, error) {
	if len(declaredOutputs) == 0 {
		return &ArtifactSet{Artifacts: []Artifact{}}, nil
	}

	// Collect all file paths from declared outputs
	var allPaths []string

	for _, output := range declaredOutputs {
		// Resolve relative to base directory
		fullPath := output
		if !filepath.IsAbs(output) {
			fullPath = filepath.Join(h.BaseDir, output)
		}

		// Check if path exists
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("declared output does not exist: %s", output)
			}
			return nil, fmt.Errorf("stat output %q: %w", output, err)
		}

		if info.IsDir() {
			// Collect all files in directory recursively
			files, err := h.collectFilesFromDir(fullPath)
			if err != nil {
				return nil, fmt.Errorf("collecting files from %q: %w", output, err)
			}
			allPaths = append(allPaths, files...)
		} else {
			// Single file
			allPaths = append(allPaths, fullPath)
		}
	}

	// Sort paths for determinism
	// CRITICAL: Do not rely on filesystem ordering
	sort.Strings(allPaths)

	// Remove duplicates (in case overlapping paths were declared)
	allPaths = deduplicateSorted(allPaths)

	// Read and normalize file contents
	artifacts := make([]Artifact, 0, len(allPaths))
	for _, path := range allPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading artifact %q: %w", path, err)
		}

		// Normalize content if normalizer is configured
		if h.Normalizer != nil {
			content = h.Normalizer.Normalize(content)
		}

		// Normalize path to forward slashes for cross-platform determinism
		normPath := filepath.ToSlash(path)

		artifacts = append(artifacts, Artifact{
			Path:    normPath,
			Content: content,
		})
	}

	return &ArtifactSet{Artifacts: artifacts}, nil
}

// collectFilesFromDir recursively collects all files in a directory.
// Returns paths sorted for determinism.
func (h *Harvester) collectFilesFromDir(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (we only want files)
		if d.IsDir() {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort for determinism
	sort.Strings(files)

	return files, nil
}

// deduplicateSorted removes duplicates from a sorted slice.
func deduplicateSorted(sorted []string) []string {
	if len(sorted) == 0 {
		return sorted
	}

	result := make([]string, 0, len(sorted))
	result = append(result, sorted[0])

	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			result = append(result, sorted[i])
		}
	}

	return result
}
