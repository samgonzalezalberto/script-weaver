## Invalidation Map Overview

An **invalidation map** is a deterministic, per-task mapping from task ID &rarr; set invalidation reasons.

* Each reason explains why a task must be re-executed or skipped
* Reasons must be **atomic, explicit, and reproducible**
* Maps are computed for each execution mode (clean, incremental, cached, parallel)

## Types of Invalidation Reasons

* **InputChanged:** Content of a declared input differs from last execution
* **EnvChanged:** Relevant environment variable(s) changed
* **DependencyInvalidated:** One or more upstream tasks were invalidated
* **GraphStructureChanged:** Task was added, removed, or connected differently
* **CommandChanged:** The command executed by the task differs
* **OutputChanged:** Declared output(s) have changed (e.g., file missing, hash mismatch pre-execution)

## Propagation Rules

* Downstream tasks inherit invalidation reasons by **referencing the root cause**
* `DependencyInvalidated` reasons must include the `SourceTaskID` of the originally invalidated task
* Aggregation order is deterministic (e.g., topological + TaskID)

## Trace Semantics

* Traces include invalidation events
* Recording is **observational only** and does not affect execution
* Trace ordering rules from Sprint-03 apply (topological + TaskID)

## Determinism Guarantees

* Given identical inputs, environment, and graph, **invalidations are identical across runs**
* Downstream aggregation of reasons is **byte-for-byte reproducible**
* Trace events are deterministic even in parallel execution

