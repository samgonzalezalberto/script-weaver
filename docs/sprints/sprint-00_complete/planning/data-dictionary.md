# Deterministic Task Execution — Data Dictionary

## Task
A declarative definition of work to be executed deterministically.

Includes:
- Inputs
- Command
- Declared environment
- Declared outputs

Excludes:
- Implicit dependencies
- External side effects

---

## Input
Any file or file set whose content contributes to task identity.

Includes:
- File contents
- Expanded, sorted paths

Excludes:
- File metadata not explicitly read

---

## Task Hash
A deterministic identifier representing the full identity of a task execution.

Includes:
- Inputs
- Command
- Environment variables
- Declared outputs
- Working directory identity

Excludes:
- Timestamps
- Machine-specific data

---

## Artifact
A file or directory produced by a task and explicitly declared in `outputs`.

Includes:
- Normalized file contents
- Stable metadata

Excludes:
- Undeclared files
- Temporary or intermediate files

---

## Cache Entry
A stored result of a task execution keyed by Task Hash.

Includes:
- stdout
- stderr
- exit code
- artifacts

Excludes:
- Execution timestamps
- Host-specific metadata

---

## Deterministic Environment
A controlled execution context where all observable inputs are explicit.

Includes:
- Declared environment variables
- Pinned tools via inputs or env

Excludes:
- Implicit system state
- External network access (unless allowed)

---

## Replay
The act of returning cached results without re-executing a task.

Includes:
- Bit-for-bit identical outputs

Excludes:
- Any execution side effects
