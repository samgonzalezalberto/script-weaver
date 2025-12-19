
# CLI Engine Notes (Determinism Boundary)

This sprint implements the CLI as a **pure deterministic boundary**.
The CLI’s only job (in this phase) is to map user-supplied arguments into a fully
canonical `CLIInvocation` structure.

## What "No Implicit State" Means Here

To avoid accidental nondeterminism, the parser is written to *not consult* any
ambient machine state:

* **No environment variables**: the parser never calls `os.Getenv` and never uses
	env-derived defaults (e.g., `DEBUG`, `CLICOLOR`).
* **No current-working-directory inference**: the parser never calls `os.Getwd`.
	Instead, `--workdir` is mandatory and must be an **absolute path**.

This is stricter than typical CLIs, but it guarantees that the same args produce
the same invocation independent of where the command is run from.

## Canonical Path Rules

All user-provided paths are canonicalized before use:

* `filepath.Clean` is applied to remove `.` and `..` components.
* Any *relative* path (`--graph`, `--cache-dir`, `--output-dir`, `--trace`) is
	resolved via `filepath.Join(WorkDir, relative)`.
* Because `WorkDir` must be absolute, this resolution never consults the process
	CWD.

## How We Prove It (Unit Tests)

The tests validate:

* identical args + identical `--workdir` produce **byte-identical** `CLIInvocation`
	structs.
* relative paths resolve under `WorkDir` even if the process CWD is changed.
* environment variables do not affect the parsed invocation.

## Exit Code Semantics

The CLI guarantees stable, semantic exit codes. These mappings are authoritative:

* **0 — Success**
    * *Returned when*: Graph execution completes successfully.
    * *Guarantees*: All requested outputs are present and valid.
    * *Determinism*: Always returned for a successful run, regardless of mode (Clean/Incremental).

* **1 — Graph Failure**
    * *Returned when*: The engine executes but fails to complete the graph (e.g., a task returns non-zero, dependency failure).
    * *Guarantees*: The failure was deterministic (not a system crash) and related to the graph logic itself. Trace is valid.
    * *Determinism*: Identical graph logic + identical inputs = identical failure node.

* **2 — Invalid Invocation**
    * *Returned when*: CLI usage is incorrect, arguments are missing, or input files are malformed (see below).
    * *Guarantees*: Execution never started; no side effects occurred.
    * *Determinism*: Parsing is pure; identical bad args always yield Code 2.

* **3 — Deterministic Configuration Error**
    * *Returned when*: Flags are syntactically valid but semantically conflicting (e.g., impossible output config).
    * *Guarantees*: Configuration is fundamentally invalid.
    * *Determinism*: Static validation rules always catch this before execution.

* **4 — Internal / System Error**
    * *Returned when*: The system encounters a non-deterministic fault (Panic, IO failure, Cache Corruption).
    * *Guarantees*: The system state may be undefined, but the process terminated safely.
    * *Determinism*: These overlap with environmental factors (disk full), so they represent the "catch-all" for nondeterminism.

## Graph Validation Failure Clarification

To ensure strict separation between "Invocation" and "Execution":

* **JSON Parse Error** (Malformed Graph File) maps to **Exit Code 2 (Invalid Invocation)**.
    * *Reason*: A malformed file implies the input itself is invalid, similar to a syntax error. The entrypoint rejects it before attempting execution.
* **Semantic Graph Error** (Cycles, Missing Tasks) maps to **Exit Code 1 (Graph Failure)**.
    * *Reason*: The input was syntactically valid, but logic failed.

## Output Isolation (Overwrite Policy)

To guarantee deterministic outputs, the CLI applies an **Overwrite** policy to
`OutputDir` on every run:

* If `OutputDir` does not exist, it is created.
* If it exists, **all children are removed** before execution starts.

This prevents stale files from previous runs (that are not regenerated this run)
from persisting and becoming observable nondeterminism.

## Clean vs Incremental Mapping

* **Incremental mode** uses the persistent filesystem cache at `CacheDir`.
* **Clean mode** uses a no-op cache (always miss, ignore writes), making the run
	indistinguishable from a fresh machine with an empty cache.

In both modes, the `OutputDir` overwrite policy ensures the on-disk outputs are
reset to exactly the files produced by the current execution.

## Trace Lifecycle

When tracing is enabled, the CLI creates a valid trace file *before* execution
and overwrites it atomically after execution completes.
If execution panics or otherwise fails before producing trace bytes, the CLI still
emits a deterministic empty trace for the computed `graphHash`.

## Persistence Strategy (Crash Safety)

### Trace

Trace files are written using an atomic temp-file + rename strategy:

1. Write the full canonical JSON bytes to `trace.json.tmp.*` in the same directory.
2. `fsync` best-effort.
3. `rename` into place.

This prevents partially-written JSON from being observed after a crash/panic.
Additionally, the CLI creates a valid (possibly empty) trace file *before* engine
execution starts so the trace artifact exists even if execution fails.

### Cache

`FileCache.Put` is implemented as a directory-level atomic commit:

1. Create a temp entry directory next to the final `{CacheDir}/{prefix}/{hash}` entry.
2. Write all artifact blobs into the temp dir.
3. Write `metadata.json` last into the temp dir.
4. Remove any existing entry directory (best-effort) and `rename` the temp dir into place.

This ensures a crash cannot leave a *corrupt* `metadata.json` in the canonical entry path.
At worst, a crash between remove/rename yields a cache miss, which is safe.

## Ambiguities Found During TDD

* Graph file format was not fully specified in Sprint-05 docs beyond “explicit path”.
	The implementation assumes a strict JSON format with top-level `tasks` and `edges`.
	Earlier sprints mention YAML or JSON; if YAML is required for Sprint-05, the loader
	should be extended with a deterministic YAML decoder.

