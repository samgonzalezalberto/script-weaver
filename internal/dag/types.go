package dag

import "scriptweaver/internal/core"

// GraphHash is the deterministic identity of a TaskGraph.
//
// It is computed solely from task definition content and dependency structure.
// It MUST be stable across different insertion orders of tasks and edges.
type GraphHash string

// TaskDefHash is the deterministic identity of a task definition as used by the DAG model.
//
// Note: this is intentionally distinct from core.TaskHash (execution/cache identity),
// since DAG identity is computed from the declarative definition fields required by
// the DAG specification prompts.
type TaskDefHash string

// Edge represents a dependency relation: To depends on From.
//
// Semantics (from spec.md): a directed edge From -> To means To can only run after
// From completes successfully.
type Edge struct {
	From string
	To   string
}

// TaskNode is an immutable node in the TaskGraph.
//
// Name is an external identifier used for addressing edges and debugging.
// The graph hash primarily derives from the task definition content and the
// canonicalized dependency structure.
type TaskNode struct {
	Name           string
	Task           core.Task
	DefinitionHash TaskDefHash
	canonicalIndex int
}

// CanonicalIndex returns the node's deterministic position in the graph's canonical ordering.
func (n *TaskNode) CanonicalIndex() int { return n.canonicalIndex }

// Hash returns the graph's stable identity.
func (h GraphHash) String() string { return string(h) }

// String returns the string representation of the TaskDefHash.
func (h TaskDefHash) String() string { return string(h) }
