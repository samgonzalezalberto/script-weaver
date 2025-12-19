# Sprint-05 Backlog & Deferred Items

This document captures work identified during Sprint-05 that was deferred to preserve scope or maintain focus on the core deterministic CLI contract.

## Deferred Features

### 1. YAML Graph Support
*   **Description**: Support for YAML-formatted graph definition files.
*   **Status**: Deferred.
*   **Rationale**: Sprint-05 implementation focused strictly on JSON validity for determinism. The loader architecture is extensible, but currently only validates and parses strict JSON.
*   **Dependency**: Requires a deterministic YAML parser/decoder to avoid ambiguity in parsing rules.

### 2. Interactive / REPL Mode
*   **Description**: Interactive shell/REPL for graph manipulation.
*   **Status**: Explicit Non-Goal (Sprint-05).
*   **Rationale**: The focus was on a pure, scriptable execution boundary. Interactive modes introduce state complexity.

### 3. Shell Auto-Completion
*   **Description**: Bash/Zsh completion scripts for CLI flags.
*   **Status**: Deferred.
*   **Rationale**: Usability enhancement, not core to correctness.

### 4. Environment Variable Configuration
*   **Description**: Inferring default values from `SCRIPT_WEAVER_` env vars.
*   **Status**: Explicitly Prohibited (Sprint-05).
*   **Rationale**: To prove determinism, we strictly enforced "No Implicit State". Future sprints may introduce a "Configuration Loader" layer that is auditable, but for now, it is blocked.

### 5. Detailed UX Polish
*   **Description**: Colored output, progress bars, human-friendly error formatting.
*   **Status**: Deferred.
*   **Rationale**: "UX polish beyond correctness" was out of scope. Current output is raw and semantic.

## Known Limitations

### 1. Mandatory Absolute WorkDir
*   **Limitation**: The CLI requires an explicit, absolute path for `--workdir`. It refuses to infer CWD.
*   **Impact**: Slightly more verbose invocation for users.
*   **Preservation**: This limitation is intentional to guarantee invariant path resolution. Future "Launcher" scripts can wrapper this.

### 2. Output Directory Overwrite
*   **Limitation**: The `OutputDir` is wiped clean at the start of every run.
*   **Impact**: Users cannot accumulate outputs from disjoint runs in the same directory without risk of data loss.
*   **Preservation**: Essential for proving "Output Purity". Accumulation logic belongs in a higher-level orchestration layer, not the core CLI.
