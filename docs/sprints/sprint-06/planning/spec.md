# Graph Definition Specification (v1)

## Overview

This document defines the canonical, versioned, deterministic graph format used by ScriptWeaver.

Graphs are declarative execution plans. They must be:

* Fully deterministic
* Strictly validated
* Hash-stable

## Format

* Encoding: JSON
* Schema: JSON Schema Draft 2020-12
* File Extension: `.graph.json`

## Top-Level Structure

Required top-level fields:

* `schema_version`
* `graph`
* `metadata`

Unknown top-level fields are rejected.

## Versioning

* `schema_version` is required
* Initial version: `1.0.0`
* Minor versions: backward-compatible
* Major versions: breaking

## Graph Object

The `graph` object defines execution structure.

Required fields:

* `nodes`
* `edges`

### Nodes

* Unique, stable `id` (string)
* Explicit `type` (string)
* Immutable `inputs` (key-value map)
* Immutable `outputs` (list of strings)

Node order is not semantically relevant.

### Edges

Edges define directed dependencies.

* Structure: List of `{from: string, to: string}` objects
* Must form a directed acyclic graph (DAG)
* No implicit dependencies

## Metadata
.

Allowed fields:

* `name`
* `description`
* `labels`

Metadata is strictly excluded from hash computation.

## Determinism Rules

The graph hash is computed from the canonical JSON representation of the `graph` object.

Normalization Rules:

1.  **Key Sorting**: All JSON object keys are sorted lexicographically.
2.  **List Sorting**:
    *   `nodes` array is sorted by `id`.
    *   `edges` array is sorted by `from`, then `to`.
    *   `outputs` array is sorted lexicographically.
3.  **Whitespace**: No whitespace in canonical form.

Excluded from Hash:

*   Top-level `metadata` object
*   Top-level `schema_version`
* Non-semantic metadata

## Validation Phases

1. Schema validation
2. Structural validation (DAG, references)
3. Semantic validation (types, inputs)

Failures halt execution.

## Rejection Rules

* Unknown fields &rarr; error
* Missing required fields &rarr; error
* Cycles &rarr; error
* Duplicate node IDs &rarr; error

## Compatibility Guarantees

* v1 graphs will always execute identically
* v1 hashes are stable forever

No implicit migrations are applied.