# Sprint-09 &mdash; Test-Driven Design

## Unit Tests

* Valid plugin manifest parses correctly
* Duplicate plugin IDs are rejected
* Unsupported hooks cause validation failure

## Integration Tests

* Engine loads plugins from expected directory
* Nested subdirectories are ignored (non-recursive discovery)
* Plugins execute at correct lifecycle hook
* Plugin execution order is deterministic
* Plugin error does not crash engine

## Negative Tests

* Invalid plugin directory is skipped
* Malformed `manifest.json` is logged and skipped
* Plugin panic is recovered
* Plugin cannot mutate forbidden state

## Acceptance Criteria

* All tests pass without feature flags
* Engine behavior is unchanged when no plugins exist