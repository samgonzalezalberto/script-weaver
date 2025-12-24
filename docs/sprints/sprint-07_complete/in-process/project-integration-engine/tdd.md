# Test-Driven Development Plan &mdash; Project Integration

## Testing Philosophy

Integration behavior must be predictable, reproducible, and isolated.

## Test Categories

### 1. Workspace Initialization Tests

* `.scriptweaver/` is created if missing
* Existing workspace is reused
* Invalid workspace structure is rejected

### 2. Zero-Config Tests

* Repo with graphs runs without configs
* Defaults applied deterministically
* No side effects outside workspace

### 3. Graph Discovery Tests

* Explicit path overrides conventions
* `graphs/` discovery works
* Ambiguous discovery fails

### 4. Determinism Tests

* Same repo state &rarr; identical behavior
* Re-run does not mutate inputs
* Workspace contents are predictable

### 5. Isolation Tests

* User files never modified
* Engine state confined to `.scriptweaver/`

### 6. Failure Tests

* Missing graphs fail cleanly
* Invalid graphs fail before execution

## Golden Fixtures

* minimal-repo/
* repo-with-graphs/
* repo-with-ambiguous-graphs/
* repo-with-invalid-workspace/

Each fixture defines expected behavior.

## Regression Policy

* Integration behavior is locked per version
* Breaking changes require spec update