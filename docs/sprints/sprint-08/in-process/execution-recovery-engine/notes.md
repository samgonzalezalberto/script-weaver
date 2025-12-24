# Notes: Execution Recovery State Models (Prompt 1)

## Design decisions

- On-disk layout (not specified in the frozen spec/TDD) is implemented as JSON files under `.scriptweaver/runs/<run-id>/`:
	- `run.json` for the `Run` record
	- `failure.json` for the `Failure` record (one terminal failure per run)
	- `checkpoints/<node-id>.json` for per-node `Checkpoint` records

- JSON decoding is strict (`DisallowUnknownFields`) to enforce the frozen schema and detect accidental drift.

- `previous_run_id` is always serialized (nullable) by using a pointer field **without** `omitempty`, producing `null` when absent.

- `Failure.node_id` is optional and is omitted when absent (`omitempty`).

## Atomic + durable state writes

- All state writes use: write temp file in same directory → `fsync(tmp)` → `rename` → `fsync(directory)`.
- Directory creation is followed by syncing the directory and its parent to reduce the chance of losing directory entries on crash/power loss.

## Assumptions & edge cases

- `Checkpoint.cache_keys` must serialize as an array (not `null`). The store ensures `[]` is written when the slice is nil.
- `checkpoints/<node-id>.json` assumes `node_id` is safe as a filename (no path separators). If future graphs allow arbitrary identifiers, this mapping needs a deterministic escaping scheme.
- `Run.status` allowed values are not defined in the frozen spec/TDD; validation currently enforces only “non-empty”.
- Execution modes accept the spec-defined values: `clean`, `incremental`, and `resume-only` (even though the current CLI only accepts `clean|incremental`).

## Checkpoint validation strategy (Prompt 2)

- Source of truth for “node success” is the node exit code (`0` == success). Non-zero exit code blocks checkpoint creation.
- “Deterministic output writes” are verified by re-harvesting the node’s declared outputs via `core.Harvester` (stable ordering + optional normalization) and hashing the resulting artifacts (path + content, length-prefixed, sha256 hex). Missing declared outputs block checkpoint creation.
- “Cache entry existence” is verified by `core.Cache.Has(taskHash)` using the node’s deterministic task hash; if absent, checkpoint creation fails.
- “Trace entry completion” is verified against the provided trace events:
	- Non-cached execution requires a `TaskExecuted` event for the node and must not include `TaskFailed`.
	- Cached execution accepts `TaskArtifactsRestored` (or `TaskExecuted`) and must not include `TaskFailed`.
- Checkpoint metadata is persisted through the recovery store, which writes JSON atomically + durably (temp + fsync + rename + dir fsync).

## Resume eligibility strategy (Prompt 3)

- “Workspace validated (no corruption)” is implemented using the existing workspace validator (`workspace.EnsureWorkspace(projectRoot)`), which rejects unauthorized entries under `.scriptweaver/`.
- Graph hash unchanged is checked by loading the referenced previous run (`previous_run_id`) and comparing `graph_hash` strings.
- `previous_run_id` must be present and must load successfully; resume is rejected otherwise.
- Run retry semantics are enforced by requiring the previous run to have a persisted `failure.json` record, and requiring `new.retry_count == prev.retry_count + 1`.
- “No upstream invalidation markers exist” is checked by computing the transitive upstream closure from the intended resume checkpoint node (including the node itself) in the current `incremental.GraphSnapshot` and ensuring none of those nodes are marked `Invalidated=true` in the current `incremental.InvalidationMap`.

## Failure taxonomy & recording (Prompt 4)

- Implemented four explicit error types mirroring the frozen spec failure classes:
	- Graph failure: schema/structural/graph load problems → `FailureClass=graph`, `Resumable=false`
	- Workspace failure: invalid `.scriptweaver` workspace or IO/config issues preventing execution → `FailureClass=workspace`, `Resumable=false`
	- Execution failure: node-level failure (non-zero exit / task marked failed) → `FailureClass=execution`, `Resumable=true` (conditionally; actual resume still requires checkpoints)
	- System failure: panic / engine internal error → `FailureClass=system`, `Resumable=true`

- CLI wiring now creates a run under `.scriptweaver/runs/<run-id>/` and writes `failure.json` for:
	- Graph load/validation failures
	- Workspace validation failures
	- Engine errors/panics
	- Any node failure detected in `GraphResult.FinalState`

- Run IDs are generated as random 128-bit hex strings (the frozen spec does not define a required format).

## Execution engine integration (Prompt 5)

### Execution modes

- CLI now accepts the spec-defined modes: `clean`, `incremental`, and `resume-only`.
- `resume-only` fails if resume is not possible (no eligible previous run + checkpoints).

### `previous_run_id` detection

- The frozen spec allows `previous_run_id` to be "supplied or detectable".
- Current implementation chooses a deterministic detected previous run by:
	- Listing `.scriptweaver/runs/*`.
	- Loading `run.json` for each run.
	- Filtering by matching `graph_hash`.
	- **Preferring only runs that have a persisted `failure.json`** (resume is only meaningful after a non-successful termination).
	- Selecting the run with the greatest `start_time` (tie-break by lexicographic `run_id`).

### Resume plan strategy

- Resume is implemented by building an `incremental.IncrementalPlan` overlay used by the DAG executor:
	- Nodes with a valid checkpoint whose `cache_keys[0]` matches the current deterministic task hash are eligible for `ReuseCache`.
	- Eligible nodes also require the corresponding cache entry to still exist; missing cache entries are treated as workspace corruption (resume rejected).
	- All other nodes are planned as `Execute`.
	- Dependency rule: a node can only be `ReuseCache` if all its upstream dependencies are also `ReuseCache` (matching the existing incremental plan semantics).

- Important interaction with output directory clearing:
	- The CLI clears the output directory at the start of each run (overwrite policy).
	- Downstream task hashes depend on upstream output content (because outputs are used as inputs).
	- Therefore, resume planning computes hashes in topological order and **restores outputs for upstream `ReuseCache` nodes before hashing downstream nodes**, ensuring task hashes are computed against the deterministic cached artifacts rather than stale/missing workspace files.

- The resume eligibility checker is enforced before applying a resume plan:
	- `previous_run_id` points to a prior failed run (must have `failure.json`).
	- `retry_count` increments from the previous run.
	- No invalidation markers exist upstream of the chosen resume checkpoint node.

### Checkpoint persistence

- Checkpoints are persisted during execution (per-node) when a task reaches a successful terminal outcome (`COMPLETED` / planned cache restore) with exit code `0`.
	- This is implemented via a DAG executor observer hook so crash recovery can safely resume from the last persisted checkpoint.
- Checkpoint persistence is gated on tracing being enabled (`--trace`), because trace completeness is a required checkpoint validity rule.


