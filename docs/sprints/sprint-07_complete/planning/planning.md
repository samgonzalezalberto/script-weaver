# Sprint-07: Project Integration Mode

## Sprint Objective

Transform ScriptWeaver from a standalone engine into a drop-in project tool that works inside existing repositories with minimal configuration.

The sprint focuses on **adoption, ergonomics, and conventions,** not new execution semantics.

## Problem Statement

Although ScriptWeaver has a frozen and deterministic graph contract (Sprint-06), it still feels like an engine that must be manually wired.

Missing today:

* A standard project layout
* Predictable workspace rules
* Zero-config defaults
* Clear boundaries between user code and ScriptWeaver state

Without these, integration friction remains high.

## Scope (In-Scope)

* Standard project integration layout
* `.scriptweaver/` workspace directory
* Convention-over-configuration directory
* Zero-config execution path
* Repo-local graph discovery rules

## Non-Goals (Explicitly Out of Scope)

* Changes to graph schema or semantics
* Failure recover or resume logic
* Plugin system
* Executor feature expansion
* Remote execution

## Deliverables

1. Defined project layout contract
2. `.scriptweaver/` workspace rules
3. Graph discovery conventions
4. Zero-config execution behavior
5. Integration documentation fixtures

## Success Criteria (Sprint Exit)

* ScriptWeaver runs inside a repo with one command
* No required configuration for default use
* Workspace is deterministic and isolated
* No violation of Sprint-06 graph contract

## Risks

* Implicit behavior creating ambiguity
* Leaking environment-specific assumptions
* Mixing user artifacts with engine state

## Mitigations

* Explicit conventions over heuristics
* Strict workspace boundaries
* Deterministic discovery rules

## Downstream Enablement

* Sprint-08: Failure Recovery & Resume
* Sprint-09: Plugin System