## GraphDelta

Represents the difference between two graph executions.

Fields:

* AddedNodes
* RemovedNodes
* ModifedNodes

## InvalidationReason

Enumerates the cause of task invalidation.

Examples:

* InputChanged
* EnvChanged
* DependencyInvalidated
* GraphStructureChanged

## NodeExecutionDecision

Represents the execution decision for a task.

Possible values:

* Execute
* ReuseCache
> **Note:** Conditional or runtime-basedping is explicitly unsupported in Sprint-02.

## IncrementalPlan

A deterministic plan describing which tasks will execute and which will be reused.

Includes:

* Ordered task list
* Execution decisions per task

## IncrementalResult

Represents the result of an incremental graph execution.

Includes:

* GraphResult
* Execution decisions
* Invalidation reasons