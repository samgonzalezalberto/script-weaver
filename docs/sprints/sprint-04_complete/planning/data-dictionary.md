## InvalidationMap

Represents the complete set of invalidation reasons for all tasks in a graph

Fields:

* TaskID &rarr; set of InvalidationREasons
* Deterministically ordered (e.g., topological + TaskID)

## InvalidationReason

Describes a single cause for task invalidation. Defined as a structured object, not just an enum.

Structure:

*   **Type** (Enum): The category of invalidation
    *   `InputChanged`
    *   `EnvChanged`
    *   `DependencyInvalidated`
    *   `GraphStructureChanged`
    *   `CommandChanged`
    *   `OutputChanged`
*   **SourceTaskID** (Optional ID): The TaskID of the root cause (required for `DependencyInvalidated`)
*   **Details** (Optional Map/String): Context specific to the type (e.g., `InputName`, `EnvName`)

## TraceEvent

* Existing field updates to include `InvalidationReasons` list

## TraceHash

* Recomputed deterministically to include invalidation reason information