## Graph Construction Tests

* Valid graph creation with single node
* Valid graph creation with multiple independent nodes
* Valid graph creation with complex dependency chains
* **Diamond Dependency Test**: A -> B, A -> C, B -> D, C -> D. D must run exactly once after B and C.

## Cycle Detection Tests

* Graph creation must fail if a cycle is present
* Self-referential dependencies must be rejected
* Indirect cycles must be detected deterministically

## Deterministic Ordering Tests

* Serial and parallel execution must yield identical results
* **Tie-Breaker Stability**: Tasks with equal priority must execute in a predictable order (e.g., lexical sort of names) when resources allow.
* Graph hash must be invariant to execution strategy (Serial vs Parallel)

## Parallel Safety Tests

* Concurrent execution must not introduce race conditions
* **Shared Resource Isolation**: Two tasks writing to the same output directory (if allowed) must be deterministic or explicitly forbidden.
* Task isolation must be preserved
* **State Transition Integrity**: A task must never be RUNNING and FAILED simultaneously.

## Cache Hit / Miss Graph Tests

* Cached nodes must not re-execute
* Mixed cache hit/miss scenarios must be deterministic
* **Partial Restoration**: If part of the graph is cached, the restored artifacts must be bit-identical to fresh execution.
* Graph-level results must be identical across repeated runs

## Failure Propagation Tests

* **Cascade Failure**: If Root fails, Dependent must be SKIPPED, not FAILED.
* **Partial Success**: Independent branches must complete even if a neighbor branch fails.

