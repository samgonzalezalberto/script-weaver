// Package core defines the domain models for deterministic task execution.
package core

import (
	"bytes"
	"regexp"
)

// DefaultNormalizer removes common nondeterministic patterns from output.
//
// From spec.md Output Determinism:
//
//	"Outputs are normalized to remove nondeterministic data (e.g., timestamps)."
//
// From tdd.md Test 6:
//
//	"Normalized cached output MUST be identical across runs."
//
// This normalizer handles:
//   - ISO 8601 timestamps (2024-12-13T10:30:45Z)
//   - Common log timestamps (2024-12-13 10:30:45)
//   - Unix timestamps (1702469445)
//   - Time durations that vary (took 1.234s)
//   - Process IDs (pid 12345)
//   - Memory addresses (0x7fff5fbff8c0)
type DefaultNormalizer struct {
	// Patterns to replace with stable placeholders
	patterns []*normPattern
}

type normPattern struct {
	regex       *regexp.Regexp
	replacement []byte
}

// NewDefaultNormalizer creates a normalizer with common patterns.
func NewDefaultNormalizer() *DefaultNormalizer {
	return &DefaultNormalizer{
		patterns: []*normPattern{
			// ISO 8601 timestamps: 2024-12-13T10:30:45Z, 2024-12-13T10:30:45.123Z
			{
				regex:       regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`),
				replacement: []byte("<TIMESTAMP>"),
			},
			// Common log timestamps: 2024-12-13 10:30:45, 2024/12/13 10:30:45
			{
				regex:       regexp.MustCompile(`\d{4}[-/]\d{2}[-/]\d{2}\s+\d{2}:\d{2}:\d{2}(\.\d+)?`),
				replacement: []byte("<TIMESTAMP>"),
			},
			// Unix timestamps (10+ digits): 1702469445, 1702469445123
			{
				regex:       regexp.MustCompile(`\b1[0-9]{9,12}\b`),
				replacement: []byte("<UNIX_TS>"),
			},
			// Duration patterns: took 1.234s, 123ms, 1.5 seconds
			{
				regex:       regexp.MustCompile(`\b\d+(\.\d+)?\s*(ms|s|seconds?|minutes?|hours?)\b`),
				replacement: []byte("<DURATION>"),
			},
			// Process IDs: pid 12345, PID: 12345, [12345]
			{
				regex:       regexp.MustCompile(`\b[Pp][Ii][Dd][:\s]*\d+\b`),
				replacement: []byte("pid <PID>"),
			},
			// Memory addresses: 0x7fff5fbff8c0
			{
				regex:       regexp.MustCompile(`0x[0-9a-fA-F]{8,16}`),
				replacement: []byte("<ADDR>"),
			},
		},
	}
}

// Normalize removes nondeterministic patterns from content.
func (n *DefaultNormalizer) Normalize(content []byte) []byte {
	result := content

	for _, p := range n.patterns {
		result = p.regex.ReplaceAll(result, p.replacement)
	}

	return result
}

// RawNormalizer performs no normalization, preserving raw bytes exactly.
// Use this when you want bit-for-bit identical output without any processing.
type RawNormalizer struct{}

// NewRawNormalizer creates a normalizer that preserves content unchanged.
func NewRawNormalizer() *RawNormalizer {
	return &RawNormalizer{}
}

// Normalize returns content unchanged.
func (n *RawNormalizer) Normalize(content []byte) []byte {
	return content
}

// StreamNormalizer normalizes line endings for cross-platform consistency.
// Converts all line endings to Unix-style (LF).
type StreamNormalizer struct {
	// Inner normalizer to apply after line ending normalization
	Inner OutputNormalizer
}

// NewStreamNormalizer creates a normalizer that standardizes line endings.
func NewStreamNormalizer(inner OutputNormalizer) *StreamNormalizer {
	return &StreamNormalizer{Inner: inner}
}

// Normalize converts CRLF to LF and optionally applies inner normalizer.
func (n *StreamNormalizer) Normalize(content []byte) []byte {
	// Convert CRLF to LF for cross-platform consistency
	result := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))

	// Apply inner normalizer if configured
	if n.Inner != nil {
		result = n.Inner.Normalize(result)
	}

	return result
}
