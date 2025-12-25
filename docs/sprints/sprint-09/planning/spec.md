# Sprint-09 &mdash; Specification

## Plugin Definition

A **plugin** is an external Go module that implements a well-defined interface and is loaded by the Script Weaver engine at runtime.

Plugins:

* Do not own core execution flow
* May observe or modify limited state via provided interfaces
* Must be explicitly invoked by the engine

## Discovery Rules

* Plugins are discovered from a fixed directory (e.g. `.scriptweaver/plugins/`)
* Each plugin resides in its own subdirectory
* Discovery is non-recursive
* Absence of plugins is valid and non-error

## Registration

* Each plugin directory MUST contain a `manifest.json` file defining the `PluginManifest`
* Registration occurs at engine startup
* Invalid plugins are skipped with logged errors
* Duplicate plugin IDs are rejected

## Lifecycle Hooks

Initial supported hooks:

* `BeforeRun`
* `AfterRun`
* `BeforeNode`
* `AfterNode`

Hooks:

* Are synchronous
* Execute in deterministic order
* Receive read-only context unless explicitly allowed

## Failure Semantics

* Plugin panics are caught
* Plugin errors are logged and surfaced
* Core execution continues unless explicitly configured otherwise