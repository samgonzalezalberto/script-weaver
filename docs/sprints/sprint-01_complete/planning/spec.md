## High-Level DAG Model

The DAG engine executes a set of tasks organized as a Directed Acyclic Graph (DAG), where nodes represent tasks and edges represent dependency relationships.

Execution proceeds only when all upstream dependencies of a task have completed successfully.

## TaskGraph Definition

A **TaskGraph** is a finite set of TaskNodes and Edges forming a directed acyclic structure. The graph represents a single deterministic execution unit.

The graph itself is treated as a first-class deterministic entity with a stable identity.

## Dependency Semantics

* A directed edge from A to B means B depends on the **successful completion** (exit code 0) of A.
* Tasks with no incoming edges are considered **roots**.
* Tasks with no outgoing edges are considered **leaves**.
* Dependencies are strict; partial completion is not allowed.

## Execution States

Each task within the graph must exist in exactly one of the following states at any given time:

1. **PENDING**: Initial state; waiting for dependencies.
2. **RUNNING**: Currently executing.
3. **COMPLETED**: Successfully finished with exit code 0.
4. **FAILED**: Finished with non-zero exit code.
5. **SKIPPED**: Did not run because an upstream dependency failed or was skipped.
6. **CACHED**: Result retrieved from cache (logically equivalent to COMPLETED).

## Failure Propagation Rules

* If a task transitions to **FAILED**:
  * All immediate and transitive downstream dependencies must transition to **SKIPPED**.
* Failure state must be deterministic and reproducible.
* The graph execution result must clearly distinguish between successful, failed, and skipped tasks.

## Cache Behavior at Graph Level

* Each TaskNode participates in cache resolution independently.
* Cached results may satisfy node execution if inputs, dependencies, and task definition are unchanged.
* **Graph-level cache identity** must be derived solely from:
  * Ordered list of Task Definitions
  * Deterministic Dependency Structure (Edge List)
  * Normalized Inputs across all tasks

## Deterministic Scheduling Guarantees

* **Parallel Execution Safety**: Parallel execution must yield bit-for-bit identical artifacts and logs as serial execution.
* **Tie-Breaking Strategy**: When multiple tasks are eligible to run (State: PENDING, all dependencies COMPLETED), the scheduler must prioritize them deterministically.
  * **Primary Key**: Topological depth (shallowest first, if BFS strategy is chosen) OR rigorous alphabetical order of Task Name.
  * *Constraint*: The specific scheduling algorithm (e.g., lexical sort of ready queue) must be fixed and documented to ensure replayability of execution logs (if logs captured order).
  * *Note*: While *outputs* must be independent of order, the *order itself* should be stable for debugging.
* **Concurrency Independence**: The system must verify that increasing worker threads does not alter the Graph Hash or Task Hashes.

