package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"scriptweaver/internal/core"
)

type edgeIndex struct {
	from int
	to   int
}

// TaskGraph is an immutable, validated DAG definition.
//
// It is safe for concurrent read access.
type TaskGraph struct {
	nodesByName map[string]*TaskNode
	nodes       []*TaskNode // canonical order

	edges []edgeIndex // sorted

	outgoing [][]int // by canonical index, sorted ascending
	incoming [][]int // by canonical index, sorted ascending
	indeg    []int   // by canonical index
	depth    []int   // by canonical index (topological depth)

	hash GraphHash
}

// NewTaskGraph builds and validates a TaskGraph.
//
// Validation runs immediately and rejects:
//   - empty or duplicate task names
//   - edges referencing unknown tasks
//   - duplicate edges
//   - self-loops
//   - any cycle (direct or indirect)
func NewTaskGraph(tasks []core.Task, edges []Edge) (*TaskGraph, error) {
	if len(tasks) == 0 {
		return nil, invalidf("no tasks")
	}

	nodesByName := make(map[string]*TaskNode, len(tasks))
	nodes := make([]*TaskNode, 0, len(tasks))

	for _, t := range tasks {
		if t.Name == "" {
			return nil, invalidf("task name is required")
		}
		if _, exists := nodesByName[t.Name]; exists {
			return nil, invalidf("duplicate task name: %q", t.Name)
		}

		defHash := computeTaskDefHash(t.Inputs, t.Env, t.Run)
		node := &TaskNode{Name: t.Name, Task: t, DefinitionHash: defHash}
		nodesByName[t.Name] = node
		nodes = append(nodes, node)
	}

	// Canonicalize nodes: sort by definition hash primarily, then by name as stable tie-breaker.
	sort.Slice(nodes, func(i, j int) bool {
		ai, aj := nodes[i], nodes[j]
		if ai.DefinitionHash != aj.DefinitionHash {
			return ai.DefinitionHash < aj.DefinitionHash
		}
		return ai.Name < aj.Name
	})
	for i, n := range nodes {
		n.canonicalIndex = i
	}

	nameToIndex := make(map[string]int, len(nodes))
	for _, n := range nodes {
		nameToIndex[n.Name] = n.canonicalIndex
	}

	// Canonicalize edges: map to indices, reject invalid, sort, reject duplicates.
	mapped := make([]edgeIndex, 0, len(edges))
	seen := make(map[edgeIndex]struct{}, len(edges))
	for _, e := range edges {
		fromNode, okFrom := nodesByName[e.From]
		toNode, okTo := nodesByName[e.To]
		if !okFrom {
			return nil, invalidf("edge references unknown task (from): %q", e.From)
		}
		if !okTo {
			return nil, invalidf("edge references unknown task (to): %q", e.To)
		}
		if fromNode.Name == toNode.Name {
			return nil, invalidf("self-loop: %q -> %q", e.From, e.To)
		}

		pair := edgeIndex{from: nameToIndex[fromNode.Name], to: nameToIndex[toNode.Name]}
		if _, exists := seen[pair]; exists {
			return nil, invalidf("duplicate edge: %q -> %q", e.From, e.To)
		}
		seen[pair] = struct{}{}
		mapped = append(mapped, pair)
	}

	sort.Slice(mapped, func(i, j int) bool {
		a, b := mapped[i], mapped[j]
		if a.from != b.from {
			return a.from < b.from
		}
		return a.to < b.to
	})

	outgoing := make([][]int, len(nodes))
	incoming := make([][]int, len(nodes))
	indeg := make([]int, len(nodes))
	for _, e := range mapped {
		outgoing[e.from] = append(outgoing[e.from], e.to)
		incoming[e.to] = append(incoming[e.to], e.from)
		indeg[e.to]++
	}
	for i := range outgoing {
		sort.Ints(outgoing[i])
	}
	for i := range incoming {
		sort.Ints(incoming[i])
	}

	g := &TaskGraph{
		nodesByName: nodesByName,
		nodes:       nodes,
		edges:       mapped,
		outgoing:    outgoing,
		incoming:    incoming,
		indeg:       indeg,
	}

	if err := g.validateAcyclic(); err != nil {
		return nil, err
	}

	g.depth = g.computeDepth()

	g.hash = g.computeGraphHash()
	return g, nil
}

// Hash returns the stable identity for this graph.
func (g *TaskGraph) Hash() GraphHash { return g.hash }

// Node returns a node by name.
func (g *TaskGraph) Node(name string) (*TaskNode, bool) {
	n, ok := g.nodesByName[name]
	return n, ok
}

// Nodes returns the nodes in canonical order.
func (g *TaskGraph) Nodes() []*TaskNode {
	out := make([]*TaskNode, len(g.nodes))
	copy(out, g.nodes)
	return out
}

// Edges returns the dependency edges as stable (From, To) name pairs in canonical order.
func (g *TaskGraph) Edges() []Edge {
	out := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		out = append(out, Edge{From: g.nodes[e.from].Name, To: g.nodes[e.to].Name})
	}
	return out
}

// Depth returns the deterministic topological depth of the given node name.
//
// Depth is defined as the length of the longest path from any root to the node.
func (g *TaskGraph) Depth(name string) (int, bool) {
	n, ok := g.nodesByName[name]
	if !ok {
		return 0, false
	}
	return g.depth[n.canonicalIndex], true
}

func (g *TaskGraph) computeDepth() []int {
	depth := make([]int, len(g.nodes))
	order := g.topoOrderIndices()
	for _, u := range order {
		maxParent := 0
		for _, p := range g.incoming[u] {
			cand := depth[p] + 1
			if cand > maxParent {
				maxParent = cand
			}
		}
		depth[u] = maxParent
	}
	return depth
}

// TopologicalOrder returns a deterministic topological ordering of task names.
//
// Since the graph is validated on construction, this method must not fail.
func (g *TaskGraph) TopologicalOrder() []string {
	order := g.topoOrderIndices()
	names := make([]string, 0, len(order))
	for _, idx := range order {
		names = append(names, g.nodes[idx].Name)
	}
	return names
}

func (g *TaskGraph) computeGraphHash() GraphHash {
	h := sha256.New()

	writeField := func(data []byte) {
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
		h.Write(lengthBytes)
		h.Write(data)
	}

	// Nodes (canonical order)
	writeField([]byte{byte(len(g.nodes))})
	for _, n := range g.nodes {
		writeField([]byte(n.DefinitionHash))
	}

	// Edges (canonical order)
	writeField([]byte{byte(len(g.edges))})
	for _, e := range g.edges {
		writeField([]byte{byte(e.from >> 24), byte(e.from >> 16), byte(e.from >> 8), byte(e.from)})
		writeField([]byte{byte(e.to >> 24), byte(e.to >> 16), byte(e.to >> 8), byte(e.to)})
	}

	sum := h.Sum(nil)
	return GraphHash(hex.EncodeToString(sum))
}
