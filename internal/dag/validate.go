package dag

import (
	"container/heap"
)

// validateAcyclic proves the graph has no cycles using Kahn's algorithm.
//
// If a cycle exists, it deterministically extracts one cycle path for error reporting.
func (g *TaskGraph) validateAcyclic() error {
	order := g.topoOrderIndices()
	if len(order) == len(g.nodes) {
		return nil
	}

	cyclePath := g.findCycleDeterministic()
	return cycleError(cyclePath)
}

type intMinHeap []int

func (h intMinHeap) Len() int           { return len(h) }
func (h intMinHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intMinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *intMinHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *intMinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// topoOrderIndices returns a deterministic topological ordering of node indices.
//
// Determinism: the ready queue is a min-heap by canonical index.
func (g *TaskGraph) topoOrderIndices() []int {
	indeg := make([]int, len(g.indeg))
	copy(indeg, g.indeg)

	ready := &intMinHeap{}
	heap.Init(ready)
	for i := range indeg {
		if indeg[i] == 0 {
			heap.Push(ready, i)
		}
	}

	out := make([]int, 0, len(indeg))
	for ready.Len() > 0 {
		n := heap.Pop(ready).(int)
		out = append(out, n)
		for _, m := range g.outgoing[n] {
			indeg[m]--
			if indeg[m] == 0 {
				heap.Push(ready, m)
			}
		}
	}
	return out
}

// findCycleDeterministic performs a deterministic DFS over canonical indices to
// extract one cycle path.
//
// This does not attempt to list all cycles; it returns a single stable witness.
func (g *TaskGraph) findCycleDeterministic() []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)

	color := make([]int, len(g.nodes))
	parent := make([]int, len(g.nodes))
	for i := range parent {
		parent[i] = -1
	}

	var cycle []int

	var dfs func(u int) bool
	dfs = func(u int) bool {
		color[u] = gray
		for _, v := range g.outgoing[u] { // already sorted
			if color[v] == white {
				parent[v] = u
				if dfs(v) {
					return true
				}
				continue
			}
			if color[v] == gray {
				// Found a back-edge u -> v. Reconstruct cycle v ... u -> v.
				cycle = append(cycle, v)
				cur := u
				for cur != -1 && cur != v {
					cycle = append(cycle, cur)
					cur = parent[cur]
				}
				cycle = append(cycle, v)
				return true
			}
		}
		color[u] = black
		return false
	}

	for i := 0; i < len(g.nodes); i++ {
		if color[i] != white {
			continue
		}
		if dfs(i) {
			break
		}
	}

	if len(cycle) == 0 {
		return nil
	}

	// cycle currently ends with v and starts with v, but is in reverse-ish parent walk order.
	// Normalize to names in forward order.
	// Example: [v, u, ..., v] where the middle is reverse. Reverse all, keep closure.
	rev := make([]int, len(cycle))
	for i := range cycle {
		rev[i] = cycle[len(cycle)-1-i]
	}

	out := make([]string, 0, len(rev))
	for _, idx := range rev {
		out = append(out, g.nodes[idx].Name)
	}
	return out
}
