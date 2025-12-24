# Sprint-07 Summary: Project Integration Mode

## Sprint Overview
**Sprint Name:** Project Integration Mode
**Intent:** Transform ScriptWeaver from a standalone engine into a drop-in project tool that works inside existing repositories with minimal configuration.
**Relationship to Sprint-06:** Preserves the frozen graph contract while wrapping it in a deterministic integration layer.

## Objectives vs Outcomes

### Original Goals
*   Establish a standard project layout.
*   Define predictable workspace rules.
*   Enable zero-config defaults.
*   Enforce clear boundaries between user code and ScriptWeaver state.

### Outcome Confirmation
*   **Met:** ScriptWeaver now supports repo-local, zero-config integration.
*   **Met:** The `.scriptweaver/` workspace is strictly defined and isolated.
*   **Met:** Graph discovery is deterministic and convention-based.

## Key Deliverables

### 1. Project Integration Contract
*   Defined in `spec.md`.
*   Establishes the project root as the anchor for all operations.

### 2. `.scriptweaver/` Workspace Rules
*   Strict validation of allowed contents (`cache/`, `runs/`, `logs/`, `config.json`).
*   Automatic initialization for zero-config runs.
*   Rejection of unauthorized files to prevent pollution.

### 3. Deterministic Graph Discovery
*   Implemented precedence order:
    1.  Explicit CLI path
    2.  `graphs/` directory
    3.  `.scriptweaver/graphs/`
*   Ambiguity handling: Fails if multiple candidates exist at the same priority level.
*   Cross-platform determinism via sorted directory iteration.

### 4. Zero-Config Execution
*   System runs with sensible defaults if no configuration is present.
*   No environment variables or global state required.

### 5. Isolation Guarantees
*   **Sandbox Guard:** Verified that execution does not mutate user files outside `.scriptweaver/`.
*   Configuration is strictly local to the workspace.

## Constraints Enforced
The following were explicitly excluded to preserve correctness and determinism:
*   **No Graph Schema Changes:** The Sprint-06 contract remains frozen.
*   **No Recovery Logic:** Deferred to Sprint-08 to ensure integration stability first.
*   **No Plugin System:** Deferred to Sprint-09 to avoid premature abstraction.
*   **No Global State:** Integration is strictly repo-local.

## Approval Status
✅ **APPROVED**
The Documentation Agent has reviewed and approved the implementation against the planning contract. The implementation adheres strictly to the specified behavior, determinism requirements, and isolation rules.

## Impact
*   **Developer Experience:** ScriptWeaver can now be dropped into a repo and run with a single command.
*   **Reliability:** Integration behavior is predictable and isolated, reducing the risk of "it works on my machine" issues.
*   **Foundation:** The stable workspace structure paves the way for robust failure recovery (Sprint-08) and plugins (Sprint-09).
