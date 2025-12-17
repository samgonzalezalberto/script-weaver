# Implementation Notes - Sprint 02: Incremental Engine

Use this file to document architectural reasoning, tradeoffs, edge cases, and proofs of correctness.

## Deterministic Invalidation (CalculateInvalidation)

### Model

To strictly follow the Subgraph Invalidation Rules in spec.md, invalidation is computed from a minimal per-node snapshot that separates:

* **Declared input set** (the declared inputs list; compared as a set)
* **Input content hash** (a precomputed deterministic summary of resolved input contents)
* **Environment variables** (declared env map)
* **Command string**
* **Declared outputs** (compared as a set)
* **Upstream dependency identity** (direct parent node names; compared as a set)

This keeps “declared inputs” distinct from “input content”, matching the spec wording.

### Algorithm

1. **Seed direct invalidations** by comparing each node in `newGraph` against the same-named node in `oldGraph`.
	- Nodes added in `newGraph` are invalidated with `GraphStructureChanged`.
	- For nodes present in both graphs, the first differing field (in spec order) sets the direct `InvalidationReason`.
2. **Propagate strictly downstream** in deterministic topological order:
	- If any upstream dependency is invalidated, the node becomes invalidated with reason `DependencyInvalidated`.
	- Direct reasons take precedence over propagated reasons.

### Correctness / Invariants

* **Strict transitivity:** by evaluating in topological order and invalidating on “any invalidated upstream”, every descendant of an invalidated node is invalidated.
* **Determinism:** upstream/outputs/declared inputs are normalized as sets (sorted + dedup) and the propagation order is a deterministic topo sort with lexical tie-breaking.

## Incremental Planning (Execute vs ReuseCache)

### Decision Rules (Binding)

Every node must be assigned exactly one of:

* `ReuseCache`
* `Execute`

There is **no** `Skip` state and no runtime-conditional logic.

A node is `ReuseCache` **if and only if** all of the following are true:

1. The node is **not invalidated** by `CalculateInvalidation`.
2. The node’s **TaskHash exists in the cache index**.
3. **All upstream dependencies** are marked `ReuseCache`.

Otherwise, it is `Execute`.

### Determinism

Decisions are computed in a deterministic topological order with lexical tie-breaking, so the “all upstream are ReuseCache” check is well-defined and stable.

## Artifact Restoration (ReuseCache ≡ Execute)

### Requirement Mapping

When a node is planned as `ReuseCache`, the workspace must be observationally identical to a fresh execution. Artifact restoration therefore:

1. Computes the expected artifact content hash from cached bytes.
2. Checks whether the workspace file exists and matches that hash.
3. If missing or mismatched, restores the artifact bytes using an **atomic write/replace** (write temp file in same directory, then rename).

### Safety

* Restoration fails hard if any artifact cannot be retrieved from cache (e.g., cached content is missing).
* After restoration completes successfully, the workspace artifact bytes match the cached bytes exactly, satisfying “byte-for-byte identical”.

## Incremental Graph Structure (Types + Stability)

### Static Graph Stability

The graph structure is static during execution. The existing DAG graph identity (`GraphHash`) is computed from canonicalized task definition hashes + canonicalized edges, so it is stable across insertion order and can be compared across runs.

### Dynamic Overlay Types

Defined as concrete types (per Sprint-02 data-dictionary.md):

* `GraphDelta` with `AddedNodes`, `RemovedNodes`, `ModifiedNodes`.
* `NodeExecutionDecision` enum: `Execute` or `ReuseCache` (no third state).
* `IncrementalPlan` containing:
	- a deterministic ordered task list (`Order`)
	- a per-task decision map (`Decisions`)

### Deterministic Plan Hashing

`IncrementalPlan.Hash()` is defined by serializing tasks in `Order` and writing `(taskName, decision)` pairs with length-prefixing before hashing, which is deterministic even though Go map iteration is not.

## Execution Orchestration (Hybrid Run)

### Overlay Model

The static DAG graph remains unchanged; `IncrementalPlan` overlays per-node decisions onto the existing executor loop.

### Executor Integration

For each ready node:

* If `DecisionExecute`: execute via the existing runner (Sprint-00 semantics: run + harvest + cache).
* If `DecisionReuseCache`: restore via `RestoreArtifacts(taskID, cacheEntry)` (Prompt 3).

### Failure Propagation

Sprint-01 failure propagation remains unchanged: if execution fails (non-zero exit) **or** cached restoration fails, the node is treated as failed and all downstream dependents are deterministically marked skipped.

### Prohibition Compliance

There is no `Skip` decision state in planning. A `ReuseCache` decision always performs restoration; it never silently skips work.
