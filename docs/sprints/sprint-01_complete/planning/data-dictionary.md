## TaskNode

A **TaskNode** represents a single deterministic task instance within a graph, including its inputs, outputs, and dependency relationships.

## Edge

An **Edge** represents a directed dependency from one TaskNode to another, enforcing execution order.

## GraphHash

A **GraphHash** is a deterministic identifier derived from the complete structure and normalized inputs of a TaskGraph:

* **Structure**: Canonical list of edges and nodes.
* **Content**: Task definitions (inputs, commands, env).
* **Identity**: Independent of wall-clock time or execution history.

## ExecutionState

**ExecutionState** represents the lifecycle state of a TaskNode or Graph. Valid states:

* **PENDING**: Waiting for dependencies.
* **RUNNING**: Actively executing.
* **COMPLETED**: Success (exit code 0).
* **FAILED**: Non-zero exit code.
* **SKIPPED**: Dependency failed.
* **CACHED**: Restored from storage.

## GraphResult

**GraphResult** captures the final deterministic outcome of a graph execution, including:
* Map of Task Name -> ExecutionState
* Map of Task Name -> Artifact Paths
* Overall Success/Failure status
* Aggregated Logs (strictly ordered by dependency topology)