// Package core defines the domain models for deterministic task execution.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// TaskHash represents a deterministic identifier for a task execution.
//
// From data-dictionary.md:
//
//	Includes: Inputs, Command, Environment variables, Declared outputs, Working directory identity
//	Excludes: Timestamps, Machine-specific data
//
// From spec.md Cache Key Definition:
//
//	Any change to these components MUST produce a different Task Hash.
type TaskHash string

// TaskHasher computes deterministic hashes for task executions.
//
// The hash computation is designed to be:
//   - Deterministic: identical inputs always produce identical hashes
//   - Content-based: uses file contents, not metadata
//   - Ordered: all components are sorted before hashing
type TaskHasher struct{}

// NewTaskHasher creates a new TaskHasher.
func NewTaskHasher() *TaskHasher {
	return &TaskHasher{}
}

// HashInput contains all components required for computing a Task Hash.
//
// From spec.md Cache Key Definition:
//   - Sorted list of input file contents
//   - Expanded and sorted input paths
//   - Task command (run)
//   - Explicit environment variables (env)
//   - Declared outputs
//   - Working directory identity
type HashInput struct {
	// Inputs is the resolved InputSet (already sorted by InputResolver).
	Inputs *InputSet

	// Command is the task's run command string.
	Command string

	// Env is the map of explicit environment variables.
	// Only these variables are visible to the task.
	Env map[string]string

	// Outputs is the list of declared output paths.
	Outputs []string

	// WorkingDir is the working directory identity.
	// This is included to ensure tasks with different working directories
	// produce different hashes even with identical other inputs.
	WorkingDir string
}

// ComputeHash computes a deterministic TaskHash from the given inputs.
//
// The hash is computed by concatenating all components in a deterministic order:
//  1. Working directory
//  2. Command
//  3. Sorted environment variables (key=value pairs)
//  4. Sorted declared outputs
//  5. For each input (already sorted): path + content
//
// All components are length-prefixed to prevent ambiguity.
//
// From tdd.md:
//   - Test 1: Identical inputs = Identical Hash
//   - Test 3: Changed content = New Hash
//   - Test 4: Changed env = New Hash
func (h *TaskHasher) ComputeHash(input HashInput) TaskHash {
	hasher := sha256.New()

	// Helper to write length-prefixed data
	writeField := func(data []byte) {
		// Write 8-byte length prefix (big-endian)
		length := uint64(len(data))
		lengthBytes := []byte{
			byte(length >> 56),
			byte(length >> 48),
			byte(length >> 40),
			byte(length >> 32),
			byte(length >> 24),
			byte(length >> 16),
			byte(length >> 8),
			byte(length),
		}
		hasher.Write(lengthBytes)
		hasher.Write(data)
	}

	// 1. Working directory identity
	writeField([]byte(input.WorkingDir))

	// 2. Command string
	writeField([]byte(input.Command))

	// 3. Environment variables - MUST be sorted for determinism
	envKeys := make([]string, 0, len(input.Env))
	for k := range input.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	// Write env count for unambiguous parsing
	writeField([]byte{byte(len(envKeys))})
	for _, k := range envKeys {
		writeField([]byte(k))
		writeField([]byte(input.Env[k]))
	}

	// 4. Declared outputs - MUST be sorted for determinism
	sortedOutputs := make([]string, len(input.Outputs))
	copy(sortedOutputs, input.Outputs)
	sort.Strings(sortedOutputs)

	// Write outputs count
	writeField([]byte{byte(len(sortedOutputs))})
	for _, out := range sortedOutputs {
		writeField([]byte(out))
	}

	// 5. Inputs - path and content for each (already sorted by InputResolver)
	inputCount := 0
	if input.Inputs != nil {
		inputCount = len(input.Inputs.Inputs)
	}
	writeField([]byte{byte(inputCount)})

	if input.Inputs != nil {
		for _, inp := range input.Inputs.Inputs {
			// Both path and content contribute to identity
			writeField([]byte(inp.Path))
			writeField(inp.Content)
		}
	}

	// Compute final hash
	sum := hasher.Sum(nil)
	return TaskHash(hex.EncodeToString(sum))
}

// String returns the string representation of the TaskHash.
func (t TaskHash) String() string {
	return string(t)
}
