# Constraints: Deterministic Developer Automation Engine

## Determinism and Reproducibility
- All task execution must be **fully deterministic**: identical inputs produce identical outputs.
- State tracking must rely on **checksums, version-pinned logic, and immutable artifacts**.
- Outputs, patches, and logs are **normalized** to remove nondeterministic elements (timestamps, random seeds, machine-specific artifacts).

## Operational Rules
- Tasks may only operate within the **defined project workspace**; external system state is immutable unless explicitly declared.
- **External non-deterministic APIs** (network calls, third-party web services) are not automatically considered deterministic; special wrappers or mocks are required for deterministic handling.
- The engine will enforce **rollback and diff-based previews** before applying changes.
- Incremental execution is permitted only when input hashes indicate no deviation from prior runs.

## Scope Limitations
- The engine will not:
  - Guarantee determinism for arbitrary external APIs outside defined control.
  - Replace AI agents for code generation or design.
  - Act as a general-purpose CI/CD system; it focuses on deterministic verification and automation of developer workflows.
  - Handle platform-level nondeterminism (e.g., OS-level concurrency anomalies) outside the defined workspace.

## Guarantees
- Deterministic execution of tasks as defined in project configuration.
- Reproducible outputs for all local and integrated workflows.
- Traceable logs and audit records for verification and rollback.