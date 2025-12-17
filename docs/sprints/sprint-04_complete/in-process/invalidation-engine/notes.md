
# Invalidation Reason Modeling & Determinism (Sprint-04)

## Scope

This note documents the implementation decisions for the Sprint-04 **InvalidationReason** model and how we guarantee determinism/canonical bytes as required by:

- `docs/sprints/sprint-04/in-process/invalidation-engine/spec.md`
- `docs/sprints/sprint-04/planning/data-dictionary.md`
- TDD references: “Single-Task Invalidation” and “Mixed Reasons”

Code location:

- `internal/incremental/invalidation.go`

## Data Model

The data dictionary defines **InvalidationReason** as a structured object, not just an enum:

- **Type**: one of `InputChanged`, `EnvChanged`, `DependencyInvalidated`, `GraphStructureChanged`, `CommandChanged`, `OutputChanged`
- **SourceTaskID**: optional; required for `DependencyInvalidated`
- **Details**: optional contextual payload (described as “map/string”)

Implementation:

- `InvalidationReasonType` is a string enum with exactly the sprint-04 reason type values.
- `InvalidationReason` includes `Type`, `SourceTaskID`, and `Details`.
- `Details` is stored as a **sorted slice of key/value pairs** (`[]InvalidationDetail`) rather than a Go map.

Rationale for `Details` representation:

- Go map iteration order is intentionally randomized; using a map directly in serialization can create non-deterministic bytes.
- Representing details as a slice allows an explicit sort order and deduplication.

## Canonicalization Rules

Canonicalization is performed before hashing/serialization so that “logically identical” reasons produce identical bytes regardless of creation order.

Rules:

- `Type` must be present.
- If `Type == DependencyInvalidated`, then `SourceTaskID` must be present.
- `Details` is normalized to a canonical form:
	- sort by `(Key, Value)`
	- deduplicate identical `(Key, Value)` pairs
	- treat empty details as absent (normalized to `nil`)

Reason-set canonicalization (`InvalidationReasons`):

- Each reason is canonicalized.
- The list is then sorted by:
	1) **Reason type order** fixed to the spec list order,
	2) `SourceTaskID` (lexical),
	3) `Details` (lexical by key/value sequence)
- Finally, identical canonical reasons are deduplicated.

This ensures that a reason set behaves like a deterministic mathematical set.

## Deterministic Serialization (Machine-Boundary Stable)

Canonical bytes are produced via a custom binary encoding (`MarshalBinary`) with:

- Explicit **field order** (never relies on reflection field ordering)
- **Length-prefixed** UTF-8 strings
- Big-endian fixed-width integers (`uint32` for lengths/counts)
- No inclusion of:
	- pointers
	- memory addresses
	- timestamps
	- runtime-dependent values
- No iteration over Go maps during serialization

Because the encoding is based only on stable primitives (strings, fixed-width integers) and explicitly sorted slices, the byte representation is stable across:

- different machines
- different CPU architectures
- different Go versions (assuming the same string contents)

Additionally, `InvalidationMap` provides a deterministic binary encoding by sorting TaskIDs before serialization and encoding each task's canonical reason set. This avoids non-deterministic Go map iteration when producing bytes for hashing or tracing.

## Root-Cause Propagation for `DependencyInvalidated`

Spec requirement: `DependencyInvalidated` must include `SourceTaskID` of the **originally invalidated** task.

Implementation behavior:

- Directly invalidated tasks are considered root causes.
- Downstream tasks inherit dependency invalidation reasons for the upstream **root causes**, not the immediate parent.
	- For a chain `A -> B -> C` where `A` is invalidated directly, both `B` and `C` receive `DependencyInvalidated(SourceTaskID=A)`.
	- For multiple upstream roots, the reasons are included separately and sorted by `SourceTaskID`.

	### Verification proof (A → B → C)

	We need to prove the invariant: downstream tasks reference the **root cause** TaskIDs via `SourceTaskID`, not just the immediate upstream.

	Let the dependency chain be `A -> B -> C`.

	Case 1: only A is directly invalidated

	- `A` has at least one *direct* reason (e.g., `InputChanged`). Therefore the algorithm records `rootSources[A] = {A}`.
	- When evaluating `B`, it sees that `A` is invalidated, so it emits `DependencyInvalidated(SourceTaskID=A)`.
		- Since `B` has no direct reasons in this case, `rootSources[B] = {A}`.
	- When evaluating `C`, it sees that `B` is invalidated and consults `rootSources[B]`.
		- It emits `DependencyInvalidated(SourceTaskID=A)`.

	Therefore `C` references `A` directly, not `B`.

	Case 2: A is directly invalidated, and B is also directly invalidated (independent failure)

	- As in Case 1, `rootSources[A] = {A}`.
	- `B` now has both:
		- a direct reason set (so `B` is itself a root cause), and
		- dependency roots inherited from `A`.
		- Therefore `rootSources[B] = {A, B}`.
	- `C` consults `rootSources[B]` and emits one dependency reason per root:
		- `DependencyInvalidated(SourceTaskID=A)`
		- `DependencyInvalidated(SourceTaskID=B)`

	This is validated by the unit test `TestCalculateInvalidation_CascadingChain_WithIndependentMidFailure_ReferencesRootCauses` in `internal/incremental/invalidation_test.go`.

## Mixed Reasons

When a task has multiple causes (e.g., input change + env change), the invalidation map contains **all** reasons, deterministically ordered.

This is verified by tests in `internal/incremental/invalidation_test.go`.

## Invalidation Semantics & Detection

This section answers: “How do we compute reasons for a given task?” in strict alignment with Sprint-04 reason types.

### Direct detection (per-task)

For each task $T$ we compute a *direct* set of reasons by comparing the old snapshot vs new snapshot:

- `InputChanged`: `InputHash` differs
- `EnvChanged`: the declared env map differs
	- Details are populated deterministically with `EnvName=<var>` for each changed key (sorted)
- `CommandChanged`: `Command` differs
- `OutputChanged`: the declared output set differs
	- Details are populated deterministically with `OutputName=<path>` for symmetric-difference entries (sorted)
- `GraphStructureChanged`: any structural mismatch, including:
	- task did not exist previously (added)
	- declared input set differs (represented as structure change for sprint-04)
	- upstream dependency set differs
	- referenced upstream task is missing in the new graph
		- details include `UpstreamTaskID=<id>` to make the cause explicit

These direct reasons are computed independently, so multiple reasons (e.g., input + env) can co-exist.

### Dependency propagation

After direct detection, downstream tasks inherit `DependencyInvalidated` reasons referencing the *root-cause* task IDs. This ensures `SourceTaskID` is always the original invalidated task, as required.

### Deterministic ordering of the reason set

The output for each task is a deterministic set:

- canonicalize each reason (including sorting Details)
- sort reasons by (type order, `SourceTaskID`, Details)
- deduplicate identical canonical reasons

This guarantees mixed reasons remain stable across runs, and that insertion/creation order never changes the serialized bytes.

## Aggregation & Deterministic Ordering (Parallel-Safe)

Sprint-04 requires that a task can be invalidated by:

- local reasons (e.g., `InputChanged`, `EnvChanged`), and
- multiple upstream roots simultaneously (`DependencyInvalidated` for multiple `SourceTaskID`s)

### Aggregation rule

For each task $T$ we compute:

- `directReasons(T)` from snapshot comparisons (local + structure checks)
- `depReasons(T)` as the set of `DependencyInvalidated(SourceTaskID=root)` for every root cause inherited from invalidated upstream dependencies

The final reason set is:

$$ reasons(T) = Canonicalize(directReasons(T) \cup depReasons(T)) $$

Where `Canonicalize` sorts and deduplicates.

### Deterministic sort key

The canonical sort key is a total order:

1) fixed **reason type order** (matches the spec list order)
2) `SourceTaskID` (lexicographic; empty sorts before non-empty)
3) `Details` list (lexicographic by `(Key,Value)` sequence)

This exactly satisfies the requirement “sorted (e.g., by Reason Type, then SourceTaskID)”.

### Why this remains identical under parallel execution

Parallel execution changes *when* upstream tasks finish, but it must not change the final computed set for a task.

We guarantee this because:

- The invalidation map computation is purely a function of the **old/new snapshots** (graph + per-task snapshots), not runtime completion.
- We process tasks in a deterministic topological order with lexical tie-breaks.
- When aggregating upstream roots, we treat them as a mathematical set:
	- collect roots into a set
	- sort them
	- emit one `DependencyInvalidated` per root
- Finally, canonicalization sorts and deduplicates the full reason set.

Therefore, even if upstream invalidations are discovered “concurrently” in a real executor, the final serialized/canonical reason list for each task is independent of discovery order.

This is validated by the unit test `TestCalculateInvalidation_Aggregation_LocalAndMultipleDependencyReasons_DeterministicOrder` in `internal/incremental/invalidation_test.go`.

## Integration with Incremental Planning

Sprint-04 requires the **InvalidationMap** to be the source of truth for the execution planner and to cover all tasks in the graph with an explicit validated/invalidated state.

Implementation integration point:

- `internal/incremental/plan.go` exposes `PlanIncremental(oldGraph, newGraph, cache)`.

Behavior:

- Computes `inv := CalculateInvalidation(oldGraph, newGraph)`.
- Builds `plan := BuildIncrementalPlan(newGraph, inv, cache)`.
- Returns both as `PlanningResult{Invalidation: inv, Plan: plan}`.

Coverage guarantee:

- `CalculateInvalidation` produces an entry for **every task** in `newGraph`.
- A task is considered **validated** when `Invalidated == false` and `Reasons` is empty.

Planning-only guarantee (no execution):

- The planner consumes only graph snapshots + cache *presence* checks (`cache.Has`).
- It does not invoke any runner or command execution.

This integration is verified by the unit test `TestPlanIncremental_ProducesInvalidationMapCoveringAllTasks` in `internal/incremental/plan_test.go`.

## Note: DeclaredInputs

Sprint-04 reason types do not include a dedicated `DeclaredInputsChanged` reason.

However, the underlying snapshot model tracks `DeclaredInputs`, and changes to it still represent an observable structural change to a task definition.
We model this as:

- `GraphStructureChanged` with a deterministic `Details` marker: `DeclaredInputs=changed`

This stays within the sprint-04 type system while preserving an explicit, reproducible explanation.

