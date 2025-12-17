// Package dag defines the deterministic domain model for ScriptWeaver's DAG engine.
//
// It is intentionally split into:
//   - Immutable graph definition (TaskGraph): tasks + dependency structure + stable GraphHash
//   - Mutable execution state (GraphState): runtime statuses and results
//
// The graph identity (GraphHash) is computed from task definition content and
// canonicalized edge structure, making it invariant to insertion order.
package dag
