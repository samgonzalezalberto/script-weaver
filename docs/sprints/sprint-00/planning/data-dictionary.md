# Data Dictionary

## Purpose

This document defines the **canonical meaning** of key system terms.

These definitions are authoritative and must remain consistent across documentation, tests, and implementation to prevent semantic drift.

---

## Task

**Definition**  
A declarative description of work to be executed.

**Includes**
- Input definitions
- Execution command
- Declared environment variables

**Excludes**
- Execution results
- Runtime state
- Side effects outside declared artifacts

---

## Input

**Definition**  
Any file content or declared value that influences task execution.

**Includes**
- File contents
- Declared environment variable values
- Task command text

**Excludes**
- File metadata such as timestamps or permissions
- Undeclared environment variables
- Ambient system state

---

## Task Hash

**Definition**  
A deterministic identifier representing the complete input state of a task.

**Includes**
- All data required to uniquely determine task execution behavior

**Excludes**
- Execution results
- Machine-specific identifiers
- Runtime metadata

---

## Artifact

**Definition**  
A persisted output produced by a task execution.

**Includes**
- Generated files
- Build outputs
- Reports or derived data products

**Excludes**
- Logs unless explicitly declared
- Temporary or intermediate files not retained

---

## Cache Entry

**Definition**  
A stored record of a completed task execution keyed by task hash.

**Includes**
- Standard output
- Standard error
- Exit code
- Generated artifacts
- Normalized execution metadata

**Excludes**
- Partial executions
- Interrupted or incomplete runs

---

## Deterministic Environment

**Definition**  
An execution context where all sources of nondeterminism are controlled or eliminated.

**Includes**
- Fixed working directory
- Explicitly declared environment variables
- Normalized locale, time, and ordering behavior

**Excludes**
- Undeclared environment variables
- External network access unless explicitly allowed
- Ambient operating system state

---

## Replay

**Definition**  
The act of returning cached task results without re-executing the task.

**Includes**
- Identical outputs and artifacts
- Identical failure behavior

**Excludes**
- Recomputing task logic
- Producing new side effects
