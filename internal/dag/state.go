package dag

// TaskState is the runtime execution state of a node.
//
// This is intentionally separated from TaskGraph, which is immutable.
//
// From sprint-01 dag-engine/spec.md:
//
//	PENDING, RUNNING, COMPLETED, FAILED, SKIPPED, CACHED
type TaskState string

const (
	TaskPending   TaskState = "PENDING"
	TaskRunning   TaskState = "RUNNING"
	TaskCompleted TaskState = "COMPLETED"
	TaskFailed    TaskState = "FAILED"
	TaskSkipped   TaskState = "SKIPPED"
	TaskCached    TaskState = "CACHED"
)

// GraphState is the mutable runtime status for a specific execution attempt.
//
// It is designed so that the same TaskGraph can be executed multiple times
// without mutating the graph definition.
type GraphState struct {
	// Status holds per-node state keyed by task name.
	Status map[string]TaskState
}
