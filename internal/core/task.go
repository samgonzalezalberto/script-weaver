// Package core defines the domain models for deterministic task execution.
//
// These structures are derived directly from the frozen specifications in
// docs/sprints/sprint-00/planning/spec.md and data-dictionary.md.
//
// Design constraints:
//   - No implied fields (e.g., creation_date) that could affect determinism
//   - All fields are explicit and observable
//   - Structures support exact serialization for hash computation
package core

// Task represents a declarative definition of work to be executed deterministically.
//
// From data-dictionary.md:
//
//	Includes: Inputs, Command, Declared environment, Declared outputs
//	Excludes: Implicit dependencies, External side effects
//
// From spec.md Task Definition Format:
//
//	Required: name, inputs, run
//	Optional: env, outputs
type Task struct {
	// Name is the logical identifier for the task.
	// Used only for user reference; does not affect task identity/hash.
	Name string `json:"name" yaml:"name"`

	// Inputs is a list of file paths or glob patterns.
	// All inputs are expanded prior to execution.
	// Expansion MUST be deterministic and strictly sorted.
	Inputs []string `json:"inputs" yaml:"inputs"`

	// Run is the command string to execute.
	// Interpreted exactly as provided.
	Run string `json:"run" yaml:"run"`

	// Env is a map of environment variables explicitly provided to the task.
	// Only variables listed here are visible to the task.
	// Optional field.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Outputs is a list of file paths or directories expected to be produced.
	// Only declared outputs are eligible for artifact capture and caching.
	// Optional field.
	Outputs []string `json:"outputs,omitempty" yaml:"outputs,omitempty"`
}
