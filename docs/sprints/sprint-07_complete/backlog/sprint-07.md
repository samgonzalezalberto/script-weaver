# Sprint-07 Backlog & Follow-Ups

## Deferred Items
The following items were intentionally excluded from Sprint-07 and are deferred to future sprints:

*   **Failure Recovery & Resume:**
    *   **Target:** Sprint-08
    *   **Context:** Logic for resuming failed runs was out of scope for the initial integration pass.
*   **Plugin System:**
    *   **Target:** Sprint-09
    *   **Context:** Extensibility hooks were excluded to ensure the core integration contract is solid first.
*   **Remote Execution:**
    *   **Target:** Future
    *   **Context:** Not currently planned for the immediate roadmap.

## Known Follow-Ups
The following are refinements or edge cases identified during implementation (see `notes.md`) that may require future attention:

*   **Symlink Handling:**
    *   **Issue:** Validation may currently reject valid symlinks for workspace subdirectories (`cache/`, etc.) depending on platform behavior.
    *   **Action:** Refine `IsDir()` checks to correctly handle symlinks if this use case becomes required.
*   **Strict JSON Validation:**
    *   **Issue:** The current JSON parser follows "last-write-wins" for duplicate keys.
    *   **Action:** Implement a streaming decoder if strict rejection of duplicate keys becomes a security or correctness requirement.
*   **Permission Error UX:**
    *   **Issue:** Permission errors (e.g., read-only repo) surface as raw OS errors.
    *   **Action:** Wrap these in friendlier, domain-specific error messages.

## Explicit Non-Backlog Items
The following are **NOT** to be added to the backlog, as they violate the core design constraints:

*   **Global Configuration:** ScriptWeaver is strictly repo-local. No global config features will be added.
*   **Graph Schema Changes:** The graph schema is frozen (Sprint-06). Any changes require a major version revision, not a backlog item.
*   **Heuristic Discovery:** Graph discovery must remain deterministic. No "fuzzy matching" or deep search heuristics will be implemented.
