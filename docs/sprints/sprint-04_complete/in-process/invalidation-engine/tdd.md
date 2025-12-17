## Single-Task Invalidation

* Given a graph with one task
* When the task input changes
* Then the invalidation map contains exactly one reason: `InputChanged`

## Multiple Independent Task Invalidations

* Given a graph with multiple independent tasks
* When several inputs change
* Then each invalidated task lists only its own input-change reasons

## Cascading Dependency Invalidation

* Given a dependency chain A &rarr; B &rarr; C
* When A is invalidated
* Then B and C inherit `DependencyInvalidated` reasons referencing A (the root cause)

## Multiple Dependency Invalidation

* Given a task C dependent on A and B
* When A and B are both invalidated (e.g., inputs changed)
* Then C inherits separate `DependencyInvalidated` reasons for A and B
* The reasons are aggregated deterministically (e.g., sorted by SourceTaskID)

## Mixed Reasons

* Given a task with multiple causes for invalidation (e.g., input + env change)
* When executed
* Then the invalidation map contains all reasons in deterministic order

## Parallel Execution Consistency

* Given a graph executed in parallel
* When upstream invalidations occur
* Then downstream reasons are identical to serial execution

## Trace Integration

* Given a graph with invalidated tasks
* When trace events are generated
* Then each event records all invalidation reasons without affecting execution

