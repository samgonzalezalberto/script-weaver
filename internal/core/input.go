// Package core defines the domain models for deterministic task execution.
package core

// Input represents a resolved file whose content contributes to task identity.
//
// From data-dictionary.md:
//
//	Includes: File contents, Expanded sorted paths
//	Excludes: File metadata not explicitly read
//
// This structure represents a single resolved input after glob expansion.
// The path is normalized and the content is read for hashing purposes.
type Input struct {
	// Path is the expanded, normalized file path.
	// Paths are sorted deterministically across runs and machines.
	Path string

	// Content is the raw file content.
	// Used for computing task identity; file metadata is excluded.
	Content []byte
}

// InputSet represents the complete set of resolved inputs for a task.
// Inputs are always maintained in sorted order by Path.
type InputSet struct {
	// Inputs is a sorted slice of resolved input files.
	// Sorting is lexicographic by Path.
	Inputs []Input
}
