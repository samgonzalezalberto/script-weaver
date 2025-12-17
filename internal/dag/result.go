package dag

import "scriptweaver/internal/core"

// GraphResult is the deterministic summary of a graph execution attempt.
//
// This intentionally includes:
//   - Final per-node states
//   - The observed execution order (useful for determinism proofs/tests)
//
// Artifact/log capture is introduced in later prompts.
type GraphResult struct {
	GraphHash GraphHash

	// FinalState is the terminal state of each node by name.
	FinalState ExecutionState

	// ExecutionOrder is the ordered list of tasks that were started (transitioned to RUNNING).
	ExecutionOrder []string

	// TaskHashes records the deterministic per-node TaskHash.
	TaskHashes map[string]core.TaskHash

	// Stdout/Stderr/ExitCode capture the node results (executed or replayed).
	Stdout   map[string][]byte
	Stderr   map[string][]byte
	ExitCode map[string]int
}
