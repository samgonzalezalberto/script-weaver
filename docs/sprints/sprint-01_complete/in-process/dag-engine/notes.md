
# DAG Engine Notes (Sprint 01)

## Prompt 1: Graph Domain Model & Structural Identity

### Goal
Implement an immutable `TaskGraph` domain model with:

- A deterministic `GraphHash` derived from task definition *content* (`inputs`, `env`, `run`) and dependency *structure* (edges).
- Immediate validation at construction time.
- Exhaustive cycle detection (including self-loops and indirect cycles).
- Strict separation between immutable graph definition and mutable execution state.

This is a determinism-critical system: the same declarative graph must yield the same identity and validation outcome regardless of insertion order.

### Architecture: Immutable Graph vs Mutable State

Implemented in:

- Immutable definition: `internal/dag.TaskGraph` and `internal/dag.TaskNode`
- Mutable runtime: `internal/dag.GraphState` and `internal/dag.TaskState`

**Reasoning**: If execution mutates the definition object (e.g., node status stored on the node itself), then concurrency and retries can change observable state depending on scheduling, which violates determinism and makes replay/debugging ambiguous. By separating state, we can safely reuse the same `TaskGraph` for multiple deterministic runs, and any nondeterminism is confined to the state layer (which is expected to vary across attempts).

### GraphHash: Canonicalization and Order-Invariance

#### What contributes to `GraphHash`
The graph identity is computed from two canonical sequences:

1. **Node sequence**: each node contributes a `TaskDefHash` computed from:
	 - Sorted `inputs` (treated as a set for identity)
	 - Sorted `env` map (by key)
	 - `run` string
2. **Edge sequence**: each edge is represented as a pair of canonical node indices `(fromIndex, toIndex)`.

The hash excludes runtime state, timestamps, and any insertion metadata.

#### Canonical ordering rule
To make the hash invariant to the order that tasks/edges are added:

- Nodes are sorted by `(TaskDefHash, Name)`.
	- `TaskDefHash` is the primary key (content-based identity).
	- `Name` is only a stable tie-breaker if two tasks have identical content.
- Edges are mapped to indices using that node ordering, then sorted lexicographically by `(fromIndex, toIndex)`.

#### Determinism proof sketch
Let $G$ be a graph defined by a set of tasks $T$ and a set of edges $E$.

- The function $H_{task}(t)$ (`TaskDefHash`) is deterministic because:
	- It sorts all unordered data (env map keys) and normalizes the inputs ordering.
	- It uses length-prefixed field encoding, preventing concatenation ambiguity.
- The canonical node ordering $C(T)$ is deterministic because sorting by `(H_{task}(t), name)` yields a unique ordering for the same multiset of tasks.
- The canonical edge list $C(E)$ is deterministic because each edge endpoint is mapped through the same deterministic index mapping induced by $C(T)$, and the resulting list is sorted.

Therefore, `GraphHash = SHA256(encode(C(T)), encode(C(E)))` is identical for any two constructions that produce the same tasks and edges, regardless of insertion order.

#### Tradeoff: task renames and duplicate task definitions
- Task names are **not** hashed as node content; they are used to reference edges and as a tie-breaker when two tasks have identical content.
- If a graph contains *two* tasks with identical `(inputs, env, run)` content, the canonical indexing becomes name-dependent (only for those duplicates). That means the `GraphHash` may differ across a pure rename even if you consider the graph “structurally isomorphic under renaming”.

This is an explicit tradeoff: we do **not** attempt full graph-isomorphism hashing (which is complex and easy to get wrong). Instead, we guarantee the required property from the prompt: insertion-order invariance for equivalent task+edge definitions.

### Cycle Detection: Exhaustive Acyclicity Proof

#### Algorithm
Validation runs during `NewTaskGraph(...)` and proves acyclicity using **Kahn’s algorithm**:

- Compute indegree for every node.
- Maintain a deterministic ready-queue of nodes with indegree 0 (min-heap by canonical index).
- Repeatedly remove a ready node, decrement indegrees of its outgoing neighbors.

If the resulting ordering contains fewer than $|V|$ nodes, then the graph contains a cycle.

#### Why it’s exhaustive
Kahn’s algorithm is complete for cycle detection in directed graphs:

- If the graph is acyclic, at least one node has indegree 0, and the algorithm can remove nodes until all are removed.
- If the algorithm terminates early, all remaining nodes have indegree $> 0$, which implies the existence of at least one directed cycle in the remaining subgraph.

This detects self-loops (rejected earlier) and all indirect cycles.

#### Deterministic error witness
After detecting a cycle (via Kahn’s shortfall), we perform a deterministic DFS over canonical indices (neighbors sorted) to extract one stable cycle path for error reporting. This does not enumerate all cycles; it provides a consistent “witness” cycle without depending on input insertion order.

### Immediate Validation on Graph Creation
All structural checks are applied in the constructor:

- Non-empty unique task names
- Edges reference existing tasks
- No duplicate edges
- No self-loops
- No cycles

This ensures that downstream components (scheduler/executor) can treat `TaskGraph` as a trusted immutable fact.

---

## Prompt 2: Deterministic Scheduling Policy

### Goal
Implement a *policy-only* scheduler that decides which tasks are eligible to run given the immutable `TaskGraph` and the current per-node `ExecutionState`.

Constraints from `spec.md` and prompt:

- The scheduler must be a pure function: same inputs -> same output list.
- Returned list order must be deterministic.
- Tie-breaking strategy: **Topological depth + lexical sort of task name**.
- Diamond convergence node becomes ready only when **all** parents are `COMPLETED` or `CACHED`.

### Policy/Mechanics Separation
The scheduler must not:

- mutate graph or state
- start tasks
- perform I/O

It only inspects:

- graph dependency structure (incoming edges)
- current status of each node

This guarantees that determinism is not coupled to runtime mechanics (worker count, goroutine timing, etc.).

### Readiness Predicate (Deterministic Gate)
A node `N` is *ready* iff:

1. `state[N] == PENDING`
2. For every incoming dependency `P -> N`, `state[P] ∈ {COMPLETED, CACHED}`

Notably:

- `FAILED` and `SKIPPED` do **not** satisfy dependencies, matching the spec semantic “depends on successful completion”.
- If a dependency is missing from the state map, it is treated as not satisfied (conservative determinism).

This readiness function is deterministic because it is a pure predicate over a finite set of stable inputs.

### Tie-Breaking: Depth then Name

#### Depth definition
We precompute a deterministic integer `depth(node)` from the graph structure:

- Roots have depth 0.
- For an edge `P -> N`, `depth(N) >= depth(P)+1`.
- We define `depth(N) = max(depth(P)+1 for all parents P)`.

This “longest-path from any root” depth is well-defined for DAGs and independent of insertion order.

#### Sorting rule
Ready tasks are sorted by:

1. ascending `depth(node)` (shallowest first)
2. ascending task `Name` (lexical)

This produces a total order on the ready set.

### Locale/Architecture Stability Proof
We must ensure the lexical tie-break is stable across OS locale settings.

- Go’s `string` comparison and `sort.Strings` are based on raw byte (UTF-8) ordering, not locale/collation rules.
- Therefore, the ordering of task names is invariant to environment variables like `LC_ALL` and to platform locale implementations.

Architectural note: we avoid any locale-dependent collation libraries and do not case-fold or normalize Unicode in scheduling. If users want “human ordering”, that belongs in UI, not in the deterministic engine.

### Diamond Dependency Correctness
In the diamond graph `A -> B`, `A -> C`, `B -> D`, `C -> D`:

- `D` is ready iff `B` and `C` are both in `{COMPLETED, CACHED}`.
- The readiness predicate quantifies over *all* incoming parents, so `D` cannot be returned early.

This is a structural guarantee that does not depend on execution concurrency.

---

## Prompt 3: State Machine & Failure Propagation

### Goal
Implement a deterministic state management module that:

- Enforces valid/atomic state transitions.
- Immediately and transitively marks downstream dependents as `SKIPPED` when a task fails.
- Treats `SKIPPED` as terminal (“finished”) for completion detection.

### State Machine (Formal)
We model each node as a finite state machine with explicit terminal states.

Allowed transitions:

- `PENDING -> RUNNING`
- `RUNNING -> COMPLETED`
- `RUNNING -> FAILED`
- `PENDING -> CACHED` (cache shortcut; success-equivalent)
- `PENDING -> SKIPPED` (failure propagation)

Terminal states:

- `COMPLETED`, `FAILED`, `CACHED`, `SKIPPED`

Disallowed transitions (examples):

- `FAILED -> RUNNING` (explicit invariant)
- `SKIPPED -> RUNNING` (finality)
- `COMPLETED -> FAILED` (would imply retroactive failure)

### Atomicity and Determinism
We cannot rely on “eventual consistency” for determinism. The executor must apply:

1. the failing node transition to `FAILED`, then
2. the full transitive `SKIPPED` propagation

as a single atomic state update **before** the scheduler is polled again.

Implementation strategy:

- Provide transition APIs that require an expected prior state (`from`) and validate `from -> to`.
- Require the caller (executor) to hold a lock for the entire update. This eliminates the race where a dependent could be marked ready while propagation is still in-flight.

### Failure Propagation (Deterministic)
When node `X` transitions to `FAILED`, we traverse all reachable downstream nodes in the graph and mark them `SKIPPED` (if they are still `PENDING`).

Determinism constraints:

- The traversal order must be stable. We traverse by canonical node indices with a min-heap ready set so the set of nodes skipped and the order we visit them does not depend on insertion order.

Correctness argument:

- Every node reachable from `X` has at least one path from `X` to that node, so it depends (directly or indirectly) on successful completion of `X`. Since `X` failed, all such nodes are not eligible to run and must become `SKIPPED`.
- Nodes not reachable from `X` are independent and must not be skipped.

### Race Detection Guard
If propagation encounters a downstream node that is already `RUNNING`, that represents a scheduling/locking bug: the dependent should not have started after an upstream failure.

We treat this as a hard invariant violation surfaced as an error, making the race observable in tests and during development.

---

## Prompt 4: Serial Execution Baseline

### Goal
Implement a deterministic `Executor` that runs the DAG in **serial** mode as a correctness baseline, while keeping the design generic enough to support parallel execution later.

Requirements:

- Main loop must: lock state → poll scheduler → execute task (abstracted) → update state.
- In serial mode, the executed order must match the scheduler’s deterministic ordering.
- Produce a `GraphResult` with per-task final states (and a deterministic execution order log for verification).

### Core Determinism Argument
Determinism hinges on two facts:

1. **Decision determinism**: `GetReadyTasks(graph, state)` is pure and returns a deterministic ordered list.
2. **State update atomicity**: state transitions are applied under a single lock, and failure propagation is applied fully before re-polling the scheduler.

Given those, serial execution is deterministic because at each iteration we select the first element of a deterministic ready list and apply deterministic state transitions.

### Locking Strategy (Serial now, Parallel later)
Even in serial, we implement the executor with an explicit mutex around shared state so the same invariants hold when we later introduce multiple workers.

We intentionally run the *task body* outside the lock:

- Lock is held only for: selecting the next ready task, transitioning it to `RUNNING`, and committing the completion/failure state.
- This avoids holding the lock during I/O/CPU work and directly generalizes to a worker-pool model.

### Abstraction Boundary: Task Execution vs Scheduling/State
The executor depends on an abstract task runner interface (single-node execution) rather than calling OS commands directly.

This isolates determinism-critical policy (scheduler/state machine) from mechanics (process execution, mocking in tests).

Parallelism extension plan (next prompt):

- Keep the same scheduler and state machine.
- Replace the serial “execute one task then loop” with a dispatcher that starts up to N tasks concurrently.
- Preserve determinism by ensuring all state mutations (including propagation) remain lock-guarded and readiness checks remain the gate.

### Correctness Proof: Serial Order Matches Scheduler
In serial mode:

- Each iteration polls `GetReadyTasks` under lock and picks the first element.
- Therefore, the run order is exactly the concatenation of the scheduler’s first-choice at each state snapshot.

The integration tests validate this by constructing a graph where new deeper tasks become ready while shallow tasks remain pending, and asserting that the executor never “jumps ahead” of the scheduler’s depth+lex priority.

---

## Prompt 5: Parallel Safety & Concurrency Invariants

### Goal
Enable concurrency > 1 while proving that increasing concurrency does not change:

- Graph identity (`GraphHash`)
- Final per-node states (`GraphResult.FinalState`)
- Deterministic “plan-visible” ordering fields in the result

Later prompts will extend this proof to artifacts/log aggregation; this step focuses on state correctness and deterministic scheduling under concurrency.

### Locking Strategy (Coarse-Grained Graph Lock)
We use a single mutex protecting the shared `ExecutionState` map and executor bookkeeping.

Rules:

- **All reads for scheduling** (`GetReadyTasks`) happen under the mutex.
- **All writes** (transitions, propagation) happen under the mutex.
- The **task body** (runner execution) happens *outside* the mutex.

This is deliberately coarse-grained for simplicity and to eliminate subtle races in a determinism-critical system.

### Deterministic Parallel Dispatch Policy
Naively, parallel dispatch can become timing-dependent: if a fast task completes early, it may unlock deeper tasks earlier, changing the observed start order across runs.

To prevent that, we adopt a **depth-staged** policy:

- We execute tasks in non-decreasing topological depth.
- For a given depth $d$, we consider tasks at depth $d$ in lexical name order and dispatch them up to the concurrency limit.
- We do not advance to depth $d+1$ until all tasks at depth $d$ are terminal (`COMPLETED`, `FAILED`, `CACHED`, `SKIPPED`).

Why this preserves determinism:

- Depth is a pure function of the immutable DAG structure.
- Within a depth, there are no intra-depth edges (because any edge increases depth by at least 1), so concurrent execution cannot violate dependency constraints.
- Dispatch order within a depth is lexical (byte-order string compare), which is locale-independent.
- Advancement to the next depth depends only on the terminality of the current depth’s tasks, which is deterministic given deterministic task outcomes.

This policy sacrifices some potential overlap (a deeper branch might be able to start while unrelated shallow roots are still running), but it guarantees stable scheduling independent of OS thread interleavings.

### Race-Freedom Argument
The critical race is: “a worker completes/fails while the scheduler is computing readiness.”

- Since readiness checks and state writes are both lock-guarded, the scheduler observes a consistent snapshot.
- On failure, `FailAndPropagate` is executed under the same lock before any subsequent scheduling decision.

### Deadlock Impossibility
We avoid deadlocks by enforcing:

- Single mutex only (no lock ordering problems).
- No blocking operations (channel receive/send, task execution) while holding the mutex.

Worker goroutines communicate completion via buffered channels; the coordinator receives completions without holding the mutex.

---

## Prompt 6: Cache Identity & Partial Restoration

### Goal
Integrate the Sprint-00 cache into the DAG execution flow with these guarantees:

- `GraphHash` identifies the plan, but cache validity is per-node via `TaskHash`.
- On a cache hit, the node transitions to `CACHED` and the task body is not executed.
- Cached nodes behave exactly like `COMPLETED` for dependency gating.
- Mixed cache states (some nodes cached, others executed) work correctly.

### Node-Level Identity: `TaskHash`
We use the existing Sprint-00 hashing definition (`core.TaskHasher`) as the node execution identity:

- Includes: resolved input file contents, command (`run`), explicit env, declared outputs, working directory identity.
- Excludes: timestamps and host metadata.

Determinism argument: the hash input is fully content-based and uses sorted order for inherently unordered components (env map keys, outputs list, input expansion).

### Cache Integration Point in the Executor
The DAG executor claims a task (moves it out of `PENDING`) and delegates execution to a cache-aware runner.

- The runner checks the cache *before* executing the task command.
- If hit: it replays artifacts and returns `FromCache=true`.
- The executor commits the node state to `CACHED` (terminal, success-equivalent).

This preserves determinism while preventing duplicate dispatch in parallel mode.

### Partial Restoration Correctness
On replay, artifacts are restored bit-for-bit from cached blobs. To ensure restored artifacts land in the correct workspace (and do not encode machine-specific absolute paths), we store artifact paths **relative to the working directory**.

Invariant:

- If an artifact was harvested from `{WorkingDir}/X`, the cache stores path `X`.
- On replay, we join `{WorkingDir}` with `X` and write the exact bytes.

This ensures:

- Correct restoration location for this run.
- Portability across machines as long as the workspace-relative layout is the same.

### CACHED Satisfies Dependencies
The scheduler’s readiness predicate and `IsSuccessful` treat `CACHED` as success-equivalent to `COMPLETED`.

Therefore, downstream nodes become ready after upstream nodes are either `COMPLETED` or `CACHED`, matching the spec.






