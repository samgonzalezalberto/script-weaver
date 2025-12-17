# Deterministic Task Execution Tool — Specification

## Purpose

This specification defines the externally observable behavior required for the tool to be considered correct.  
The tool executes user-defined tasks in a deterministic manner such that identical inputs always produce identical outputs and cached results may be safely replayed.

This document defines *what must be true*, not how it is implemented.

---

## Task Definition Format

Tasks are defined declaratively using a structured configuration format (e.g., YAML or JSON).

Each task definition MUST include the following fields:

### Required Fields

- **name**
  - Logical identifier for the task.
  - Used only for user reference.

- **inputs**
  - A list of file paths or glob patterns.
  - All inputs are expanded prior to execution.
  - Expansion MUST be deterministic and strictly sorted.

- **run**
  - A command string to execute.
  - Interpreted exactly as provided.

### Optional Fields

- **env**
  - A map of environment variables explicitly provided to the task.
  - Only variables listed here are visible to the task.

- **outputs**
  - A list of file paths or directories expected to be produced by the task.
  - Only declared outputs are eligible for artifact capture and caching.

---

## Deterministic Guarantees

The tool MUST guarantee the following:

1. **Input Determinism**
   - Input files are read by content.
   - Glob expansion is strictly sorted.
   - File ordering is stable across runs and machines.

2. **Environment Determinism**
   - Only explicitly declared environment variables are visible.
   - Tool versions and binaries are the user’s responsibility and MUST be pinned via:
     - Inputs, or
     - Explicit environment variables.
   - No implicit tool discovery is permitted.

3. **Execution Determinism**
   - Tasks execute in an isolated, controlled environment.
   - External network access is disabled unless explicitly allowed.

4. **Output Determinism**
   - Outputs are normalized to remove nondeterministic data (e.g., timestamps).
   - File ordering and metadata are stable.

---

## Cache Key Definition

Each task execution produces a **Task Hash**, which is computed from:

- Sorted list of input file contents
- Expanded and sorted input paths
- Task command (`run`)
- Explicit environment variables (`env`)
- Declared outputs
- Working directory identity

Any change to these components MUST produce a different Task Hash.

---

## Cache Behavior

- If a Task Hash has been seen before:
  - The task MUST NOT be re-executed.
  - Cached results are replayed exactly.

- Cached data includes:
  - stdout
  - stderr
  - exit code
  - artifacts defined by `outputs`

---

## Failure Behavior

- Failed executions (non-zero exit code) are cacheable.
- Replaying a failed task MUST return:
  - Identical stdout
  - Identical stderr
  - Identical exit code
- Failed tasks MUST NOT partially update artifacts.
