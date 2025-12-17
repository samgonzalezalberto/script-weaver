# Sprint-01 Backlog

## Deferred Features
These features were identified during planning or implementation but explicitly scoped out of Sprint-01 to prioritize strict determinism.

*   **Graceful Cancellation**: Handling `SIGINT` / `Context` cancellation to stop the DAG cleanly. Currently, the executor runs to completion or crash.
*   **Incremental Rebuilds**: The engine currently rebuilds the entire graph plan every time. Usage of the cache is node-local, but the graph structure itself is not differentially updated.
*   **Distributed Execution**: The system is designed for single-machine execution. Multi-machine coordination is out of scope.
*   **Plugin System**: No external extensions or dynamic loading of task types.
*   **Observability Layers**: Advanced tracing, metrics, or structured event logs (beyond basic execution sorting).

## Known Limitations
Trade-offs made deliberately to enforce correctness invariants.

*   **Depth-Staged Parallelism**: The scheduler waits for *all* tasks at Depth `N` to complete before starting *any* task at Depth `N+1`. This limits maximum theoretical concurrency but guarantees deterministic Execution Log order.
*   **Coarse-Grained Locking**: The `Executor` uses a single global mutex for all state transitions. This prevents race conditions but limits throughput scaling on very large graphs (10k+ nodes).
*   **Strict Renaming Sensitivity**: `GraphHash` includes task names as tie-breakers. Renaming a task without changing its content *will* change the Graph Identity. We do not support "isomorphic graph detection".
*   **Memory Usage**: The entire graph structure and state map are held in memory.

## Explicitly Postponed Ideas
Concepts discussed but shelved for future consideration.

*   **Interactive Mode**: A TUI or adjusting the plan mid-flight.
*   **Fine-Grained Locking**: Moving to per-node locks or actor-based state management.
*   **Custom Scheduling Policies**: Allowing users to choose "Maximize Concurrency" vs "Deterministic Logs" via configuration.
*   **Remote Caching**: The current cache design implies a local store; remote artifact storage protocols are not yet defined.
