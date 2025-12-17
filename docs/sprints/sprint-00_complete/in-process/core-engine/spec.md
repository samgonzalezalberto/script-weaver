# Specification

## Purpose

This document defines the **functional and behavioral requirements** that must be true for the deterministic task execution tool to be considered correct.

Any implementation that satisfies this specification is valid. Any behavior not explicitly defined here is considered undefined.

---

## Core Concept

The system executes user-defined tasks in a controlled environment such that:

**Identical inputs always produce identical observable results.**

This includes task identity, execution behavior, outputs, and cached results.

---

## Task Definition Format

Tasks are defined declaratively using either **YAML or JSON**.

### Top-Level Structure

A task definition file contains a mapping of task names to task definitions.

Example (illustrative only):

    tasks:
      build:
        inputs:
          - src/**
          - package.json
        run: "command to execute"
        env:
          KEY: value

---

## Required Task Fields

### inputs (required)

A list of file paths or glob patterns that define the complete input set for the task.

Rules:
- Paths are resolved relative to a fixed working directory.
- Globs are expanded deterministically.
- File contents, not timestamps or metadata, define input identity.

---

### run (required)

A command string representing the task’s execution logic.

Rules:
- The command string is treated as immutable input.
- Any change to the command string invalidates prior cache entries.

---

### env (optional)

A map of environment variables explicitly provided to the task.

Rules:
- Only declared variables are visible during execution.
- Undeclared environment variables must not influence execution.
- Any change to declared variables invalidates prior cache entries.

---

## Deterministic Guarantees (Hard Requirements)

The system **must guarantee** the following:

1. **Task Identity Determinism**  
   A task’s identity is a pure function of:
   - Input file contents
   - Task command
   - Declared environment variables
   - Referenced tool versions (if applicable)

2. **Execution Isolation**  
   Tasks execute in an environment where:
   - Undeclared environment variables are inaccessible
   - External network access is disabled unless explicitly allowed
   - Working directory and filesystem view are controlled

3. **Output Stability**  
   All observable outputs are normalized to remove nondeterministic data, including:
   - Timestamps
   - Random ordering
   - Unspecified locale or timezone effects

4. **Repeatability**  
   Re-running a task with identical inputs must produce byte-for-byte identical results or replay cached results.

---

## Cache Behavior

### Cache Key

Each task execution is keyed by a **task hash**, computed from:
- Sorted input file contents
- Task command
- Declared environment variables
- Execution metadata required for determinism

---

### Cache Contents

For each task hash, the cache stores:
- Standard output
- Standard error
- Exit code
- Generated artifacts
- Normalized execution metadata

---

### Cache Replay Rules

- If a matching task hash exists, execution must be skipped.
- Cached results must be returned exactly as stored.
- Cache replay must be indistinguishable from live execution.

---

## Failure Behavior

### Execution Failures

If a task fails:
- The failure result is cached.
- Replaying the task returns the same failure result.

---

### Invalid Tasks

If a task definition is malformed:
- The task must not execute.
- A deterministic error must be returned.

---

### Partial Execution

If execution is interrupted:
- No cache entry may be written.
- The task must be treated as never executed.

---

## Out of Scope

This specification explicitly does **not** define:
- User interfaces
- Distributed execution
- Remote caching
- Scheduling or orchestration semantics
