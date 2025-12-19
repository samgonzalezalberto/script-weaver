## CLI as Execution Boundary

The CLI is the **sole authority** for:

* Declaring execution intent
* Supplying configuration and inputs
* Selecting execution mode (clean vs incremental)
* Defining output locations

All engine execution must originate from the CLI.

## Invocation Model

A CLI invocation defines:

* Graph source (explicit path)
* Working directory (Mandatory, Root for relative paths)
* Cache directory
* Output directory
* Execution mode flags
* Trace output destination

All paths must be explicit and resolved deterministically.

## Input Semantics

* All inputs must be explicitly declared
* No implicit environment variables are read
* No current-working-directory inference is allowed
* Relative paths are resolved deterministically from invocation root

## Output Semantics

The CLI must deterministically produce:

* Final GraphResult
* Trace artifact (if enabled)
* Cache updates
* Exit code

No output may depend on runtime timing or host environment.

## Exit Codes

Exit codes are **semantic**, not incidental:

* 0: Success
* 1: Graph failure
* 2: Invalid invocation
* 3: Deterministic configuration error
* 4: Internal/System error

Identical executions must always yield identical exit codes.

## Obsesrvability Guarantees

* CLI output must be explainable by the execution trace
* Removing the CLI layer must not alter engine semantics
* CLI logic must be observational, not transformational

