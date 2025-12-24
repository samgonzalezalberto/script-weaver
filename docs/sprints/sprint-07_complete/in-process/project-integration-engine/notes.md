## Workspace init/validation edge cases (Dec 22, 2025)

- **Permissions**: Creating `.scriptweaver/` or its required subdirs (`cache/`, `runs/`, `logs/`) will fail if the repo is read-only or the process lacks permissions; these errors currently surface directly from `os.Mkdir`/`os.MkdirAll`/`os.Stat`.

- **Path collision**: If `<root>/.scriptweaver` exists as a file (not a directory), initialization rejects the workspace ("workspace path exists but is not a directory").

- **Required names as files**: If `.scriptweaver/cache` (or `runs`, `logs`) exists but is not a directory, initialization rejects the workspace as invalid.

- **Unauthorized entries**: Any top-level file/dir in `.scriptweaver/` other than `cache/`, `runs/`, `logs/`, or optional `config.json` causes immediate rejection, even if the required dirs are present.

- **Symlinks**: If `cache/`, `runs/`, or `logs/` are symlinks, `os.ReadDir`'s `DirEntry.IsDir()` may report false (depending on platform/filesystem and whether the entry type is known without `Info()`), which can cause validation to reject a symlinked directory.

## Config.json parsing strictness

- Parsing is **strict by default**: any unknown top-level field causes an error (no silent ignores).
- Only `graph_path` is permitted; attempts to set `workspace_path` or `semantic_overrides` are explicitly rejected.
- The config is **optional**: missing `.scriptweaver/config.json` returns `(ok=false)` and does not fail initialization.
- JSON must be an object (e.g., arrays/scalars fail unmarshal into an object map).
- Duplicate JSON keys are not detected with the current approach (`encoding/json` map unmarshal keeps the last value); detecting duplicates would require a streaming decoder.

## Graph discovery determinism 

- Directory iteration uses `os.ReadDir`, but the filesystem order is not relied upon.
- Entries are converted to names and sorted via `sort.Strings` before filtering candidates, ensuring consistent behavior across OS/filesystems.
- Ambiguity errors list candidate paths in sorted order.

## Isolation / sandbox guard observations

- The orchestration flow is read-only outside `.scriptweaver/`.
- A sandbox guard is implemented by snapshotting hashes of regular files outside `.scriptweaver/` before/after orchestration and failing if any changes are detected.
- No state leaks outside `.scriptweaver/` were observed in the zero-config integration tests.