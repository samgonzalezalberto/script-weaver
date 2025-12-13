// Package core defines the domain models for deterministic task execution.
package core

// Artifact represents a file or directory produced by a task
// and explicitly declared in outputs.
//
// From data-dictionary.md:
//
//	Includes: Normalized file contents, Stable metadata
//	Excludes: Undeclared files, Temporary or intermediate files
//
// Only files declared in Task.Outputs become artifacts.
// Content is normalized to remove nondeterministic data.
type Artifact struct {
	// Path is the declared output path, normalized.
	Path string

	// Content is the normalized file content.
	// Timestamps and other nondeterministic data are stripped.
	Content []byte
}

// ArtifactSet represents the complete set of artifacts produced by a task.
// Artifacts are maintained in sorted order by Path for determinism.
type ArtifactSet struct {
	// Artifacts is a sorted slice of produced artifacts.
	// Sorting is lexicographic by Path.
	Artifacts []Artifact
}
