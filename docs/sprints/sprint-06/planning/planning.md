# Sprint 06: Graph Definition & Schema Stabilization

## Sprint Objective

Freeze the external graph contract so ScriptWeaver execution is deterministic, tool-safe, and CI-safe.

This sprint defines *what a graph is* in an immutable, machine-verifiable way.

## Problem Statement

ScriptWeaver assumes a perfectly formed graph but currently lacks:

* A canonical schema
* Deterministic parsing rules
* Hash-stable normalization
* Strict validation boundaries

Without these, external adoption risks semantic drift, nondeterminism, and fragile tooling.

## Scope (In-Scope)

* Canonical graph definition format
* JSON Schema v1
* Deterministic parsing and normalization rules
* Hash stability contract
* Validation layer (schema + structural)
* Versioning and compatibility rules
* Golden fixtures and regression tests

## Non-Goals (Explicitly Out of Scope)

* Project auto-discovery
* Repository integration
* Resume / retry semantics
* Plugin system
* CLI UX polish beyond validation errors

## Deliverables

1. `graph.schema.json` (v1)
2. Parser normalization rules
3. Validation error taxonomy
4. Hash computation contract
5. Compatibility & migration policy
6. Test fixtures and hash regression tests

## Success Criteria (Sprint Exit)

* Identical graphs always produce identical hashes
* Invalid graphs fail deterministically before execution
* External tools can generate graphs safely
* Schema is frozen and versioned

## Risks

* Over-flexible schema introducing ambiguity
* Underspecified defaults causing nondeterminism
* Premature extensibility

## Mitigations

* Prefer explicit over implicit
* Reject unknown fields
* No optional semantics without defaults

## Downstream Enablement

* Sprint-07: Project Integration Mode
* Sprint-08: Failure Recovery & Resume
* Sprint-09: Plugin System
