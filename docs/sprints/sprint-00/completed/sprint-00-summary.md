# Sprint 00 Summary: Core Determinism Rules

**Status:** Completed
**Date:** 2025-12-13
**Frozen References:** `spec.md`, `tdd.md`, `data-dictionary.md`

## 1. Completed Core Modules

The following components have been implemented, audited, and verified against the frozen specifications:

| Module | File | Purpose | Audit Status |
| :--- | :--- | :--- | :--- |
| **Domain Models** | `task.go`, `input.go`, `artifact.go` | Core data structures (InputSet, ArtifactSet) enforcing sorted order and content-based identity. | ✅ Verified |
| **Resolver** | `resolver.go` | Deterministic glob expansion, path normalization, and input deduplication. | ✅ Verified |
| **Hasher** | `hasher.go` | Computation of `TaskHash` from inputs, command, env, and outputs. | ✅ Verified |
| **Executor** | `executor.go` | Process isolation, environment allowlisting, and signal handling. | ✅ Verified |
| **Harvester** | `harvester.go` | Capturing ONLY declared outputs, sorting paths, ignoring undeclared files. | ✅ Verified |
| **Normalizer** | `normalizer.go` | Scrubbing timestamps, PIDs, addresses; unifying line endings (CRLF→LF). | ✅ Verified |
| **Cache** | `cache.go` | Storage of execution results (blobs + metadata); handling of failure states. | ✅ Verified |
| **Replayer** | `replay.go` | Bit-for-bit restoration of artifacts and output streams from cache. | ✅ Verified |
| **Runner** | `runner.go` | Orchestration of the entire lifecycle (Resolve → Hash → Cache/Exec → Store). | ✅ Verified |

## 2. TDD Verification Status

All behavioral tests defined in `tdd.md` are covered by unit tests in `internal/core`:

-   **Test 1 (Stable Hash):** Covered by `hasher_test.go`.
-   **Test 2 (Cache Replay):** Covered by `cache_test.go` and `runner_test.go`.
-   **Test 3 (Input Sensitivity):** Covered by `hasher_test.go`.
-   **Test 4 (Env Sensitivity):** Covered by `hasher_test.go`.
-   **Test 5 (Isolation):** Covered by `executor_test.go`.
-   **Test 6 (Normalization):** Covered by `normalizer_test.go`.
-   **Test 7 (Bit-Perfect Replay):** Covered by `replay_test.go`.
-   **Test 8 (Output Capture):** Covered by `harvester_test.go`.
-   **Test 9 (Glob Determinism):** Covered by `resolver_test.go`.

## 3. Determinism Guarantees

The current codebase enforces strict "Level 1" determinism:

1.  **Input Stability:** All file inputs are read, sorted, and hashed by content.
2.  **Environment Zeroing:** Processes run in an empty environment by default; only strictly declared variables are passed.
3.  **Cross-Platform Paths:** All paths are normalized to forward slashes internally.
4.  **Output Stability:** Output streams are normalized for line endings and volatile data (timestamps) before caching.
5.  **Failure Determinism:** Failed tasks are cached and replayed identically, ensuring debugging is reproducible.

## 4. Freeze Confirmation

The code in `internal/core` is now considered **FROZEN** for Sprint 00.
-   No new features will be added to the core logic.
-   Development in Sprint 01 will focus on the CLI wrapper (`cmd/scriptweaver`), configuration parsing, and integration.
-   Any logic changes to core requires a formal spec update and re-audit.
