# Deterministic Task Execution — Test Design Document

## Purpose

This document defines observable behavioral tests that prove determinism.  
All tests are black-box and implementation-agnostic.

Passing these tests is required for correctness.

---

## Test 1 — Same Inputs Produce Same Hash

Given:
- Identical task definition
- Identical input file contents
- Identical environment variables

Then:
- The computed Task Hash MUST be identical.

---

## Test 2 — Same Hash Prevents Re-execution

Given:
- A previously executed task
- An identical Task Hash

Then:
- The task MUST NOT execute again.
- Cached results MUST be replayed.

---

## Test 3 — Input Content Change Invalidates Hash

Given:
- A single input file content change

Then:
- The Task Hash MUST change.
- The task MUST re-execute.

---

## Test 4 — Environment Variable Change Invalidates Hash

Given:
- A change to any declared environment variable

Then:
- The Task Hash MUST change.
- The task MUST re-execute.

---

## Test 5 — Undeclared Environment Variables Are Invisible

Given:
- An environment variable not listed in `env`

Then:
- The task MUST NOT observe it.
- The Task Hash MUST be unaffected.

---

## Test 6 — Output Normalization Removes Timestamps

Given:
- Task output containing timestamps or nondeterministic metadata

Then:
- Normalized cached output MUST be identical across runs.

---

## Test 7 — Cache Replay Is Bit-for-Bit Identical

Given:
- Cached execution results

Then:
- stdout, stderr, exit code, and artifacts MUST match exactly on replay.

---

## Test 8 — Artifact Harvesting

Given:
- Declared outputs
- Generated files within those paths

Then:
- Only declared outputs are captured.
- Artifacts are stored and replayed from cache.

---

## Test 9 — Glob Expansion Is Strictly Sorted

Given:
- Inputs defined using glob patterns

Then:
- Expanded file list MUST be strictly sorted.
- Different filesystem ordering MUST NOT affect hashing or execution.
