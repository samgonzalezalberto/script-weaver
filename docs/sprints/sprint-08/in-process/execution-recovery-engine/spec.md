# Execution Recovery & Resume Specification

## Definitions

* **Execution Run**: One attempt to execute a graph under a fixed graph hash.
* **Checkpoint**: A durable, validated execution boundary.
* **Failure**: Any non-successful termination of a run.
* **Resume**: Continuing execution from a valid checkpoint without re-executing prior work.

## Failure Classes (Frozen)

### 1. Graph Failure

* Schema violation
* Structural invalidity
* Hash mismatch
* Not resumable

### 2. Workspace Failure

* Invalid workspace structure
* Unauthorized mutation
* Not resumable

### 3. Execution Failure

* Node execution error
* Tool invocation error
* IO failure during node execution
* Conditionally resumable

### 4. System Failure

* Crash
* SIGTERM
* Power loss
* Resumable if checkpoints exist

## Checkpoint Rules

A checkpoint is valid only if:

* Node execution completed successfully
* Outputs were written deterministically
* Cache entries are present and verified
* Trace entry is complete

Checkpoint data MUST be written atomically.

## Resume Eligibility Rules

Resume is allowed if ALL conditions hold:

* Graph hash unchanged
* Workspace intact and validated
* No invalidation markers upstream of checkpoint
* Resume mode explicitly (default behavior)
* `previous_run_id` is supplied or detectable

Otherwise, execution restarts from scratch.

## Run Retry Semantics

* Retries are deterministic
* No implicit retries
* Retry count is explicit and bounded
* Each retry produces a new run ID
* Cache reuse is allowed only if inputs are identical

## Execution Modes (Expanded)

* `clean`: ignore all checkpoints
* `incremental` (default): resume where possible
* `resume-only`: fail if resume is not possible
