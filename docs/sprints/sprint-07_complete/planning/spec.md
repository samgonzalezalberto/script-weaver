# Project Integration Specification (v1)

## Overview

This document defines how ScriptWeaver integrates into an existing project repository.

Integration is **convention-based, deterministic, and local.**

## Project Root Assumption

* ScriptWeaver is invoked from a project root
* The project root is the working directory

No global state is permitted

## Workspace Directory

### `.scriptweaver/`

A reserved directory at the project root.

Purpose:

* Isolate ScriptWeaver state
* Prevent pollution of user project files

### Allowed Contents

* `cache/`
* `runs/`
* `logs/`
* `config.json` (optional)

No other directories or files are permitted.

## Graph Discovery Rules

Discovery order (precedence):

1. Explicit CLI path (if provided)
2. `graphs/` directory at project root
3. `.scriptweaver/graphs/`

Rules:

* All graphs must conform to Sprint-06 schema
* Discovery order is deterministic (first match wins)
* Ambiguous discovery (multiple candidates at same priority) fails

## Zero-Config Mode

If no configuration is provided:

* Defaults are applied
* Workspace is auto-created
* Graphs are discovered via conventions

Zero-config must never alter execution semantics.

## Configuration Rules

* Configuration is optional
* Configuration cannot override graph semantics
* Configuration affects only integration behavior

## Determinism Guarantees

* Same repo + same graph &rarr; same behavior
* No environment-derived defaults
* No hidden state outside `.scriptweaver/`

## Rejection Rules

* Multiple matching graphs &rarr; error
* Invalid workspace structure &rarr; error
* Non-conforming graphs &rarr; error
