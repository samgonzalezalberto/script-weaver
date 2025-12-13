# Sprint 00 Backlog: Deferred Items & Ideas

**Date:** 2025-12-13
**Context:** Items identified during Sprint 00 (Planning/Audit) but deferred to future sprints.

## 1. Feature Extensions (Defer to Sprint 01+)

### Parallel Execution
-   **Idea:** Run independent tasks concurrently.
-   **Requirement:** Need a DAG (Directed Acyclic Graph) executor. Currently, `Runner` handles single tasks. Dependencies must be strictly defined in `spec.md` before implementation.

### Incremental Builds & Dependencies
-   **Idea:** Allow tasks to depend on artifacts from other tasks.
-   **Requirement:** `Inputs` will need to support referencing other task outputs (e.g., `task:build:dist/`).

### Cross-Machine Reproducibility
-   **Idea:** Ensure hashes match across different absolute paths (e.g., /home/alice vs /home/bob).
-   **Status:** Partially addressed by `WorkingDir` relativity, but absolute path handling in debugging/logs needs review.

### Plugin System
-   **Idea:** Allow custom Normalizers or Executors via Wasm or shared objects.
-   **Impact:** High complexity; strictly out of scope for Sprint 00.

## 2. Technical Debt & Improvements

### Enhanced Normalization
-   **Observation:** Current `DefaultNormalizer` uses strict regexes.
-   **Idea:** Allow user-defined regex replacements in Task definition (e.g., `normalization: { patterns: [...] }`).
-   **Benefit:** Users can mask application-specific nondeterminism (e.g., random UUIDs in logs) without changing code.

### Deterministic Logging
-   **Observation:** The tool's own logs (debug/info) might contain timestamps.
-   **Requirement:** Ensure the CLI output *itself* is deterministic when requested (e.g., for verifying the tool's behavior).

### Edge Cases from Audit
1.  **Windows Path Lengths:** `FileCache` relies on filesystem paths. Deeply nested artifacts + hash prefixes might hit MAX_PATH on Windows.
    -   *Mitigation:* Consider a flat content-addressable storage (CAS) if this becomes an issue.
2.  **Large Artifacts:** `os.ReadFile` loads artifacts into memory.
    -   *Mitigation:* Switch `Runner` and `Cache` to use `io.Reader`/`io.Writer` streaming for large files in future sprints.
3.  **Symlinks:** `Harvester` currently behavior with symlinks is implicit (likely follows or copies).
    -   *Action:* Explicitly define symlink behavior in `spec.md` (likely "resolve and copy target content" for strict determinism).
