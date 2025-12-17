# Sprint-02 Backlog

**Path:** `docs/sprints/sprint-02/backlog/sprint-02.md`

## Deferred Work & Non-Goals

The following items were explicitly identified as out-of-scope for Sprint-02 in `planning.md` and `spec.md`, but represent potential future work areas.

### Execution Features
*   **Distributed or Remote Execution:** The current engine is strictly local. Support for remote workers or distributed caching is deferred. (Ref: `planning.md` Non-Goals)
*   **Speculative Execution:** No heuristics or predictive execution were implemented. The engine waits for strict dependency resolution. (Ref: `planning.md` Non-Goals)
*   **Incremental Graph Definition:** The graph structure itself must be fully defined prior to execution. Support for dynamic graph modification during runtime is pending. (Ref: `planning.md` Non-Goals)

### Observability & DX
*   **UI / CLI Enhancements:** No changes were made to the user interface to visualize cache hits/misses. (Ref: `planning.md` Non-Goals)
*   **Logging & Tracing:** Advanced telemetry for cache performance or tracing invalidation chains is not yet implemented. (Ref: `planning.md` Non-Goals)

### Optimization
*   **Performance Tuning:** Optimization beyond strict correctness requirements was deferred. This includes potential parallel artifacts download or advanced compression strategies for the cache. (Ref: `planning.md` Non-Goals)

## Ambiguities & Clarifications

Items identified during the documentation audit that may require future refinement:

*   **Cache Content Verification:** While `notes.md` specifies checking for content hash matches during restoration, the specific error handling policy for *corrupted* local files during a `ReuseCache` hit could be further detailed in future operational specs.
*   **Large Artifact Handling:** The current spec assumes atomic copy is sufficient. For very large artifacts, streaming or linking strategies may be needed in future sprints.

## Unfinished Improvements

*   **Plugin/Extension System:** The incremental engine is currently a closed system. Hooks for external validators or custom invalidation logic are backlog items. (Ref: `planning.md` Non-Goals)
