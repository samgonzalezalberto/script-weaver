## Incremental No-Op Graph

* Given a graph where all tasks are unchanged
* When executed  incrementally
* Then zero tasks are executed
* And all results are restored from cache

## Single-Node Invalidation

* Given a graph with multiple tasks
* When a single task input changes
* Then that task and all downstream tasks are re-executed
* And all upstream tasks are reused from cache

## Cascading Invalidation

* Given a dependency chain A &rarr; B &rarr; C
* When A is invalidated
* Then B and C must also be invalidated

## Parallel Graph Partial Reuse

* Given a graph with parallel branches
* When one branch is invalidated
* Then unaffected branches must be reused from cache
* And their output artifacts must be restored prior to downstream execution

## Mixed Cached and Executed Runs

* Given a graph with partial cache hits
* When executed incrementally
* Then final GraphResult must be identical to clean run

## Incremental vs Clean Equivalence

* Given identical inputs and graph structure
* When executed incrementally and clean
* Then GraphResult objects must be byte-for-byte identical
