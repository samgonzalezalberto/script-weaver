##  CLIInvocation

Represents a fully specified execution request.

Includes:

* GraphPath
* WorkDir
* CacheDir
* OutputDir
* ExecutionMode
* TraceConfig

## ExecutionMode

Enumerates allowed execution modes.

Examples:

* Clean
* Incremental

## CLIResult

Represents the observable result of a CLI execution.

Includes:

* ExitCode
* GraphResult
* TraceHash (if emitted)
* ArtifactPaths

## ExitCode

Enumerates semantic exit statuses.

Examples:

* 0 (Success)
* 1 (GraphFailure)
* 2 (InvalidInvocation)
* 3 (ConfigurationError)
* 4 (InternalError)

## PathResolution

Represents the deterministic resolution of user-provided paths.

Includes:

* OriginalPath
* ResolvedPath
* ResolutionRoot

