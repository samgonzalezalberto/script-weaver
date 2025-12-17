package dag

import (
	"container/heap"
	"fmt"
)

// IsTerminal reports whether the state is terminal (finished).
func IsTerminal(s TaskState) bool {
	switch s {
	case TaskCompleted, TaskFailed, TaskSkipped, TaskCached:
		return true
	default:
		return false
	}
}

// IsSuccessful reports whether the state satisfies dependencies.
func IsSuccessful(s TaskState) bool {
	switch s {
	case TaskCompleted, TaskCached:
		return true
	default:
		return false
	}
}

// Transition performs an atomic validated transition for a single task.
//
// The caller supplies the expected prior state (from) to make races observable.
// This function mutates the provided state map if and only if the transition is valid.
func Transition(state ExecutionState, taskName string, from, to TaskState) error {
	cur, ok := state[taskName]
	if !ok {
		return fmt.Errorf("unknown task in state: %q", taskName)
	}
	if cur != from {
		return fmt.Errorf("invalid transition for %q: expected %s, got %s", taskName, from, cur)
	}
	if !isAllowedTransition(from, to) {
		return fmt.Errorf("disallowed transition for %q: %s -> %s", taskName, from, to)
	}
	state[taskName] = to
	return nil
}

func isAllowedTransition(from, to TaskState) bool {
	switch from {
	case TaskPending:
		return to == TaskRunning || to == TaskCached || to == TaskSkipped
	case TaskRunning:
		return to == TaskCompleted || to == TaskFailed
	default:
		return false
	}
}

// FailAndPropagate transitions taskName from RUNNING to FAILED and immediately
// and transitively marks all downstream dependents as SKIPPED.
//
// Determinism:
//   - The set of nodes marked SKIPPED is defined purely by reachability.
//   - Traversal is in deterministic canonical index order.
//
// Safety:
//   - If a downstream node is already RUNNING, this is treated as an invariant
//     violation (it indicates a missing synchronization/locking bug).
func FailAndPropagate(g *TaskGraph, state ExecutionState, taskName string) error {
	if g == nil {
		return fmt.Errorf("nil graph")
	}
	node, ok := g.nodesByName[taskName]
	if !ok {
		return fmt.Errorf("unknown task: %q", taskName)
	}

	cur, ok := state[taskName]
	if !ok {
		return fmt.Errorf("unknown task in state: %q", taskName)
	}
	if cur != TaskRunning && cur != TaskFailed {
		return fmt.Errorf("cannot fail %q from state %s", taskName, cur)
	}
	if cur == TaskRunning {
		state[taskName] = TaskFailed
	}

	start := node.canonicalIndex
	visited := make([]bool, len(g.nodes))
	visited[start] = true

	hq := &intMinHeap{}
	heap.Init(hq)
	for _, d := range g.outgoing[start] {
		heap.Push(hq, d)
	}

	for hq.Len() > 0 {
		u := heap.Pop(hq).(int)
		if visited[u] {
			continue
		}
		visited[u] = true

		name := g.nodes[u].Name
		st, ok := state[name]
		if !ok {
			return fmt.Errorf("missing state for %q", name)
		}

		switch st {
		case TaskPending:
			state[name] = TaskSkipped
		case TaskRunning:
			return fmt.Errorf("invariant violation: downstream task %q is RUNNING during failure propagation", name)
		default:
			// Terminal or non-pending (e.g., already skipped). Leave unchanged.
		}

		for _, v := range g.outgoing[u] {
			if !visited[v] {
				heap.Push(hq, v)
			}
		}
	}

	return nil
}
