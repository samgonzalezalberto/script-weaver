package dag

import (
	"sort"
)

// ExecutionState maps task name to its current TaskState.
//
// It is intentionally a plain map so the scheduler can remain a pure function
// without coupling to an executor implementation.
type ExecutionState map[string]TaskState

// GetReadyTasks returns the deterministically ordered list of task names that are
// eligible to run.
//
// Policy:
//   - A task is ready iff it is PENDING and all its dependencies are COMPLETED or CACHED.
//   - The returned list is sorted by (topological depth asc, task name asc).
//
// This function is pure: it does not mutate graph or state.
func GetReadyTasks(g *TaskGraph, state ExecutionState) []string {
	if g == nil {
		return nil
	}

	ready := make([]string, 0)
	for _, node := range g.nodes {
		st, ok := state[node.Name]
		if !ok || st != TaskPending {
			continue
		}

		idx := node.canonicalIndex
		depsOK := true
		for _, parentIdx := range g.incoming[idx] {
			parentName := g.nodes[parentIdx].Name
			pst, ok := state[parentName]
			if !ok || (pst != TaskCompleted && pst != TaskCached) {
				depsOK = false
				break
			}
		}
		if depsOK {
			ready = append(ready, node.Name)
		}
	}

	sort.Slice(ready, func(i, j int) bool {
		a, b := ready[i], ready[j]
		ad, _ := g.Depth(a)
		bd, _ := g.Depth(b)
		if ad != bd {
			return ad < bd
		}
		return a < b
	})

	return ready
}
