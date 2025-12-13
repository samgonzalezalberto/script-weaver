# Core Engine Implementation Notes

This file tracks implementation decisions and evolves during coding.

---

## Phase 3, Prompt 1: Core Domain Models

**Date:** 2024-12-13

### Implemented

Created `internal/core/` package with the following domain models:

| File          | Type                      | Spec Reference                 |
| ------------- | ------------------------- | ------------------------------ |
| `task.go`     | `Task`                    | spec.md#Task-Definition-Format |
| `input.go`    | `Input`, `InputSet`       | data-dictionary.md#Input       |
| `artifact.go` | `Artifact`, `ArtifactSet` | data-dictionary.md#Artifact    |
| `doc.go`      | Package documentation     | —                              |

### Task Fields (from spec.md)

**Required:**
- `Name` — logical identifier (user reference only)
- `Inputs` — file paths or glob patterns
- `Run` — command string

**Optional:**
- `Env` — explicit environment variables map
- `Outputs` — declared output paths

### Determinism Constraints Applied

1. **No implied fields added** — No `creation_date`, `modified_at`, or similar
2. **Content-only identity** — `Input` stores content, not metadata
3. **Explicit declaration** — Only declared outputs become artifacts
4. **Sorted ordering** — `InputSet` and `ArtifactSet` maintain sorted order

### Warnings: Determinism Pitfalls

- **Do NOT** add timestamp fields to any core model
- **Do NOT** read file metadata (mtime, permissions) unless explicitly required
- **Do NOT** add machine-specific data to structures used in hashing
- Glob expansion must be sorted lexicographically before use

### Scope Reminders

- These models define *data shape*, not behavior
- Hash computation is a separate concern (future prompt)
- Cache storage is a separate concern (future prompt)

### Pointers to Data Dictionary

- **Task**: "A declarative definition of work to be executed deterministically"
- **Input**: "Any file or file set whose content contributes to task identity"
- **Artifact**: "A file or directory produced by a task and explicitly declared in outputs"

---

## Phase 3, Prompt 2: Input Resolution & Glob Expansion

**Date:** 2024-12-13

### Implemented

Created `internal/core/resolver.go` with `InputResolver` type.

| File               | Type            | Spec Reference                                       |
| ------------------ | --------------- | ---------------------------------------------------- |
| `resolver.go`      | `InputResolver` | spec.md#Deterministic-Guarantees (Input Determinism) |
| `resolver_test.go` | Tests           | tdd.md#Test-9                                        |

### Resolution Process

1. Each pattern is expanded using `filepath.Glob`
2. All expanded paths are collected into a set (deduplication)
3. Paths are normalized to forward slashes (cross-platform)
4. **Paths are explicitly sorted lexicographically** (CRITICAL for Test 9)
5. File contents are read (content-based identity, not metadata)

### Determinism Guarantees Verified

From spec.md:
- ✅ Input files are read by content
- ✅ Glob expansion is strictly sorted
- ✅ File ordering is stable across runs and machines

From tdd.md#Test-9:
- ✅ Expanded file list is strictly sorted
- ✅ Different filesystem ordering does NOT affect results

### Tests Implemented

| Test                                          | Verifies                                |
| --------------------------------------------- | --------------------------------------- |
| `TestResolve_StrictlySorted`                  | tdd.md#Test-9 - explicit sorting        |
| `TestResolve_ContentBasedIdentity`            | Content read, not metadata              |
| `TestResolve_DeterministicAcrossRuns`         | Multiple runs produce identical results |
| `TestResolve_DeduplicatesOverlappingPatterns` | No duplicate inputs                     |
| `TestResolve_EmptyPatterns`                   | Empty input handling                    |
| `TestResolve_SkipsDirectories`                | Only files, not directories             |
| `TestResolve_NormalizesPathSeparators`        | Cross-platform path normalization       |

### Key Implementation Decisions

1. **Explicit sort.Strings()** — Do NOT rely on OS directory ordering
2. **Map-based deduplication** — Overlapping patterns don't produce duplicates
3. **Forward slash normalization** — `filepath.ToSlash()` for cross-platform determinism
4. **Directories skipped** — Only regular files contribute to task identity
5. **Content via os.ReadFile** — No metadata (mtime, permissions) read

### Warnings: Determinism Pitfalls

- **Do NOT** use `range` over map directly for ordering (Go maps are unordered)
- **Do NOT** rely on `filepath.Glob` return order (OS-dependent)
- **Do NOT** include file metadata in identity (mtime changes on copy)

---

## Phase 3, Prompt 3: Task Hash Computation

**Date:** 2024-12-13

### Implemented

Created `internal/core/hasher.go` with `TaskHasher` type.

| File             | Type                                  | Spec Reference                |
| ---------------- | ------------------------------------- | ----------------------------- |
| `hasher.go`      | `TaskHasher`, `TaskHash`, `HashInput` | spec.md#Cache-Key-Definition  |
| `hasher_test.go` | Tests                                 | tdd.md#Test-1, Test-3, Test-4 |

### Hash Components (from spec.md)

The Task Hash is computed from (in order):

1. Working directory identity
2. Command string (`run`)
3. Sorted environment variables (`env`) — key-value pairs
4. Sorted declared outputs
5. For each input: path + content

### Determinism Techniques

- **SHA256** — Cryptographic hash for collision resistance
- **Length-prefixed fields** — Prevents ambiguous concatenation
- **Explicit sorting** — Env vars and outputs sorted before hashing
- **Hex encoding** — Produces 64-character lowercase hex string

### TDD Tests Verified

| Test                                             | TDD Reference | Status |
| ------------------------------------------------ | ------------- | ------ |
| `TestComputeHash_IdenticalInputsProduceSameHash` | tdd.md#Test-1 | ✅      |
| `TestComputeHash_ContentChangeInvalidatesHash`   | tdd.md#Test-3 | ✅      |
| `TestComputeHash_EnvChangeInvalidatesHash`       | tdd.md#Test-4 | ✅      |

### Additional Tests

| Test                                              | Verifies                  |
| ------------------------------------------------- | ------------------------- |
| `TestComputeHash_CommandChangeInvalidatesHash`    | Command in hash           |
| `TestComputeHash_OutputsChangeInvalidatesHash`    | Outputs in hash           |
| `TestComputeHash_WorkingDirChangeInvalidatesHash` | Working dir in hash       |
| `TestComputeHash_EnvOrderDoesNotAffectHash`       | Env sorting works         |
| `TestComputeHash_OutputsOrderDoesNotAffectHash`   | Output sorting works      |
| `TestComputeHash_InputPathChangeInvalidatesHash`  | Path is part of identity  |
| `TestComputeHash_Deterministic`                   | 100 runs = identical hash |
| `TestComputeHash_HashFormat`                      | Valid 64-char hex         |

### Prohibited Behaviors (Not Implemented)

Per instruction:
- ❌ No autodiscovery of inputs or tool versions
- ❌ No filesystem mtime/ctime usage
- ❌ No pass-through of host environment variables
- ❌ No guessing — invalid spec would fail hard

### Key Design Decisions

1. **Length-prefixed serialization** — Each field is prefixed with 8-byte length to prevent "ab"+"c" == "a"+"bc" collisions
2. **Counts included** — Number of env vars, outputs, inputs written to disambiguate empty vs missing
3. **InputSet assumed pre-sorted** — InputResolver already sorts; hasher trusts this
4. **Nil-safe** — Handles nil InputSet gracefully

---

## Phase 3, Prompt 4: Execution Isolation & Environment

**Date:** 2024-12-13

### Implemented

Created `internal/core/executor.go` with `Executor` type.

| File               | Type                          | Spec Reference                                           |
| ------------------ | ----------------------------- | -------------------------------------------------------- |
| `executor.go`      | `Executor`, `ExecutionResult` | spec.md#Deterministic-Guarantees (Execution Determinism) |
| `executor_test.go` | Tests                         | tdd.md#Test-5                                            |

### Environment Isolation (CRITICAL)

**ALLOWLIST approach** — environment starts EMPTY:

```go
cmd.Env = buildIsolatedEnv(task.Env)  // NOT os.Environ()
```

From spec.md Environment Determinism:
> "Only explicitly declared environment variables are visible."

From tdd.md Test 5:
> "An environment variable not listed in env — the task MUST NOT observe it."

### ExecutionResult Structure

| Field      | Type       | Description                     |
| ---------- | ---------- | ------------------------------- |
| `Stdout`   | `[]byte`   | Captured standard output        |
| `Stderr`   | `[]byte`   | Captured standard error         |
| `ExitCode` | `int`      | Process exit code (0 = success) |
| `Hash`     | `TaskHash` | Hash used for this execution    |

### TDD Tests Verified

| Test                                     | TDD Reference                 | Status |
| ---------------------------------------- | ----------------------------- | ------ |
| `TestExecute_UndeclaredEnvVarsInvisible` | tdd.md#Test-5                 | ✅      |
| `TestExecute_HostEnvCompletelyIsolated`  | tdd.md#Test-5 (comprehensive) | ✅      |

### Additional Tests (19 total)

| Test                                     | Verifies                 |
| ---------------------------------------- | ------------------------ |
| `TestExecute_OnlyDeclaredEnvVarsVisible` | Declared vars work       |
| `TestExecute_NoPathMeansNoPath`          | PATH not passed through  |
| `TestExecute_ExplicitPathWorks`          | Explicit PATH works      |
| `TestExecute_HomeNotPassedThrough`       | HOME not leaked          |
| `TestExecute_UserNotPassedThrough`       | USER not leaked          |
| `TestExecute_CapturesStdout`             | stdout capture           |
| `TestExecute_CapturesStderr`             | stderr capture           |
| `TestExecute_CapturesExitCode`           | Exit code 0              |
| `TestExecute_CapturesNonZeroExitCode`    | Exit code 42             |
| `TestExecute_UsesWorkingDir`             | Working directory set    |
| `TestExecute_ContextCancellation`        | Process killed on cancel |
| `TestExecute_NilTaskFails`               | Nil task rejected        |
| `TestExecute_EmptyRunFails`              | Empty run rejected       |

### Prohibited Behaviors (Verified NOT Implemented)

Per instruction:
- ❌ No autodiscovery of inputs or tool versions
- ❌ No filesystem mtime/ctime usage
- ❌ **No pass-through of host environment variables** (HOME, USER, PATH, etc.)
- ❌ No guessing — nil task and empty run fail hard

### Key Implementation Details

1. **Process groups** — `Setpgid: true` enables killing entire process tree on cancellation
2. **Shell execution** — `sh -c` interprets command strings
3. **Empty env = truly empty** — `buildIsolatedEnv({})` returns `[]string{}`, not `nil`
4. **Fail fast** — Nil task and empty run return errors immediately

### Warnings: Determinism Pitfalls

- **Do NOT** use `os.Environ()` anywhere in execution path
- **Do NOT** assume PATH exists — task must declare it
- **Do NOT** pass through "harmless" vars like TERM or LANG — they affect output

---

## Phase 3, Prompt 5: Output Normalization & Artifact Harvesting

**Date:** 2024-12-13

### Implemented

Created `internal/core/harvester.go` and `internal/core/normalizer.go`.

| File                 | Type                                                     | Spec Reference                            |
| -------------------- | -------------------------------------------------------- | ----------------------------------------- |
| `harvester.go`       | `Harvester`, `ArtifactSet`                               | spec.md#Output-Determinism, tdd.md#Test-8 |
| `normalizer.go`      | `DefaultNormalizer`, `RawNormalizer`, `StreamNormalizer` | tdd.md#Test-6                             |
| `harvester_test.go`  | Tests                                                    | tdd.md#Test-8                             |
| `normalizer_test.go` | Tests                                                    | tdd.md#Test-6                             |

### Harvester: Artifact Collection

**CRITICAL: Only declared outputs are captured** (tdd.md#Test-8)

```go
harvester.Harvest([]string{"output.txt", "build/"})
```

The harvester:
1. Resolves each declared output path relative to BaseDir
2. For files: collects the single file
3. For directories: recursively collects ALL files within
4. Sorts all paths for determinism
5. Deduplicates overlapping declarations
6. Reads content and optionally normalizes

### What Harvester Does NOT Do

- ❌ Does NOT scan for "all modified files"
- ❌ Does NOT use `git status`
- ❌ Does NOT capture undeclared files
- ❌ Does NOT read file metadata (mtime/ctime)

### Normalizer: Removing Nondeterministic Data

From spec.md Output Determinism:
> "Outputs are normalized to remove nondeterministic data (e.g., timestamps)."

From tdd.md Test-6:
> "Normalized cached output MUST be identical across runs."

**DefaultNormalizer patterns:**

| Pattern        | Example                | Replacement   |
| -------------- | ---------------------- | ------------- |
| ISO 8601       | `2024-12-13T10:30:45Z` | `<TIMESTAMP>` |
| Log timestamp  | `2024-12-13 10:30:45`  | `<TIMESTAMP>` |
| Unix timestamp | `1702469445`           | `<UNIX_TS>`   |
| Duration       | `1.234s`, `500ms`      | `<DURATION>`  |
| PID            | `pid 12345`            | `pid <PID>`   |
| Memory address | `0x7fff5fbff8c0`       | `<ADDR>`      |

**Normalizer types:**

| Type                | Behavior                                     |
| ------------------- | -------------------------------------------- |
| `DefaultNormalizer` | Replaces common nondeterministic patterns    |
| `RawNormalizer`     | Pass-through, no changes                     |
| `StreamNormalizer`  | Converts CRLF→LF, optionally chains to inner |

### TDD Tests Verified

| Test                                      | TDD Reference | Status |
| ----------------------------------------- | ------------- | ------ |
| `TestHarvest_OnlyDeclaredOutputsCaptured` | tdd.md#Test-8 | ✅      |
| `TestHarvest_DoesNotUseGitStatus`         | tdd.md#Test-8 | ✅      |
| `TestNormalization_IdenticalAcrossRuns`   | tdd.md#Test-6 | ✅      |

### Additional Tests

**Harvester tests (9):**
- `TestHarvest_DirectoryRecursive` — nested files collected
- `TestHarvest_SortedOrder` — deterministic ordering
- `TestHarvest_MissingOutputFails` — fail if declared output missing
- `TestHarvest_DeduplicatesOverlapping` — no duplicate artifacts
- `TestHarvest_NormalizesPathSeparators` — cross-platform paths

**Normalizer tests (13):**
- ISO 8601, log timestamps, Unix timestamps
- Durations, PIDs, memory addresses
- Deterministic output verification
- Stream normalizer CRLF conversion

### Prohibited Behaviors (Verified NOT Implemented)

Per instruction:
- ❌ No autodiscovery — only declared outputs
- ❌ No mtime/ctime — content only
- ❌ No git status scanning
- ❌ No guessing — missing output fails hard

---

## Phase 3, Prompt 6: Cache Storage & Replay

**Date:** 2024-12-13

### Implemented

Created `internal/core/cache.go` and `internal/core/replay.go`.

| File             | Type                                              | Spec Reference         |
| ---------------- | ------------------------------------------------- | ---------------------- |
| `cache.go`       | `Cache`, `FileCache`, `MemoryCache`, `CacheEntry` | spec.md#Cache-Behavior |
| `replay.go`      | `Replayer`, `ReplayResult`                        | tdd.md#Test-7          |
| `cache_test.go`  | Tests                                             | tdd.md#Test-2, Test-7  |
| `replay_test.go` | Tests                                             | tdd.md#Test-7          |

### Cache Interface

```go
type Cache interface {
    Has(hash TaskHash) (bool, error)
    Get(hash TaskHash) (*CacheEntry, error)
    Put(entry *CacheEntry) error
}
```

### CacheEntry Structure

From data-dictionary.md:
> Includes: stdout, stderr, exit code, artifacts
> Excludes: Execution timestamps, Host-specific metadata

```go
type CacheEntry struct {
    Hash      TaskHash         // Key
    Stdout    []byte           // Captured stdout
    Stderr    []byte           // Captured stderr
    ExitCode  int              // Process exit code
    Artifacts []CachedArtifact // Output files
}
```

### Cache Implementations

| Type          | Storage       | Use Case                       |
| ------------- | ------------- | ------------------------------ |
| `FileCache`   | Filesystem    | Persistent across runs         |
| `MemoryCache` | In-memory map | Testing, short-lived processes |

**FileCache directory structure:**
```
{CacheDir}/
  {hash[0:2]}/         # Prefix for sharding
    {hash}/
      metadata.json    # stdout, stderr, exit_code, artifact paths
      artifacts/
        0.blob         # First artifact content
        1.blob         # Second artifact content
```

### Replay: Bit-for-Bit Identical

From tdd.md Test-7:
> "stdout, stderr, exit code, and artifacts MUST match exactly on replay."

The Replayer:
1. Receives a CacheEntry
2. Restores each artifact to its original path
3. Returns stdout, stderr, exit code exactly as cached
4. Creates parent directories as needed

### TDD Tests Verified

| Test                                    | TDD Reference | Status |
| --------------------------------------- | ------------- | ------ |
| `TestCache_SameHashPreventsReExecution` | tdd.md#Test-2 | ✅      |
| `TestCache_ReplayBitForBitIdentical`    | tdd.md#Test-7 | ✅      |
| `TestReplay_BitForBitIdentical`         | tdd.md#Test-7 | ✅      |

### Additional Tests

**Cache tests (10):**
- `TestCache_FailedExecutionsCacheable` — non-zero exit codes cached
- `TestMemoryCache_IsolatesMutations` — copies prevent modification
- `TestFileCache_PersistsToFilesystem` — directory structure verified
- `TestCache_NoTimestampsStored` — excludes timestamps per spec

**Replay tests (9):**
- `TestReplay_RestoresArtifacts` — files written to workspace
- `TestReplay_FailedTaskReplay` — failed tasks replay correctly
- `TestReplay_CreatesParentDirectories` — nested paths work
- `TestReplay_OverwritesExistingFiles` — cached content replaces existing

### Key Implementation Details

1. **Hash-based sharding** — First 2 chars of hash as directory prefix
2. **Artifact blobs** — Content stored separately from metadata
3. **Deep copy** — MemoryCache copies entries to prevent mutation
4. **No timestamps** — CacheEntry has no time-related fields

### Prohibited Behaviors (Verified NOT Implemented)

Per instruction and data-dictionary.md:
- ❌ No execution timestamps in cache
- ❌ No host-specific metadata
- ❌ No mtime/ctime usage

---

## Phase 3, Prompt 7: Failure Handling & Runner Orchestration

**Date:** 2024-12-13

### Implemented

Created `internal/core/runner.go` — the orchestrator that ties all components together.

| File             | Type                  | Spec Reference                                   |
| ---------------- | --------------------- | ------------------------------------------------ |
| `runner.go`      | `Runner`, `RunResult` | spec.md#Failure-Behavior, spec.md#Cache-Behavior |
| `runner_test.go` | Tests                 | tdd.md#Test-2                                    |

### Runner: Full Execution Orchestration

The Runner coordinates:
1. Input resolution (InputResolver)
2. Hash computation (TaskHasher)
3. Cache lookup (Cache)
4. Execution (Executor) — only if cache miss
5. Artifact harvesting (Harvester) — only on success
6. Cache storage (Cache)
7. Replay (Replayer) — on cache hit

### Failure Behavior (CRITICAL)

From spec.md Failure Behavior:
> "Failed executions (non-zero exit code) are cacheable."

From prompt:
> "A failed task is deterministic! The failure must be reproducible."

**Key behaviors:**

1. **Failed tasks ARE cached** — exit code, stdout, stderr stored
2. **Failed tasks have NO artifacts** — partial outputs NOT captured
3. **Replayed failures are identical** — same stdout, stderr, exit code
4. **Cache hit skips execution** — even for failures

```go
// Only harvest artifacts on success
if execResult.ExitCode == 0 {
    artifacts, err = r.harvester.Harvest(task.Outputs)
}
```

### RunResult Structure

```go
type RunResult struct {
    Stdout    []byte
    Stderr    []byte
    ExitCode  int
    Hash      TaskHash
    Artifacts ArtifactSet  // Empty for failed tasks
    FromCache bool         // true if replayed from cache
}
```

### TDD Tests Verified

| Test                                | TDD Reference | Status |
| ----------------------------------- | ------------- | ------ |
| `TestRunner_CacheHitSkipsExecution` | tdd.md#Test-2 | ✅      |

### Additional Tests (9 total)

| Test                                      | Verifies                             |
| ----------------------------------------- | ------------------------------------ |
| `TestRunner_FailedTasksCacheable`         | Non-zero exits are cached            |
| `TestRunner_FailedTaskReplayIdentical`    | stdout, stderr, exit match on replay |
| `TestRunner_FailedTaskNoPartialArtifacts` | No artifacts for failed tasks        |
| `TestRunner_SuccessfulTaskHasArtifacts`   | Artifacts cached on success          |
| `TestRunner_CacheHitSkipsExecution`       | tdd.md#Test-2                        |
| `TestRunner_FailureIsDeterministic`       | Multiple runs = identical failure    |
| `TestRunner_ValidatesTask`                | Nil/empty task rejected              |
| `TestRunner_CleanArtifacts`               | Artifact cleanup works               |
| `TestRunner_ReplayRestoresArtifacts`      | Artifacts restored on cache hit      |

### Key Implementation Details

1. **Atomic caching** — Only cache after all operations succeed
2. **No partial artifacts** — Failed tasks cache with empty Artifacts slice
3. **FromCache flag** — Distinguishes fresh execution from replay
4. **CleanArtifacts** — Removes existing output files/dirs before execution

### Execution Flow

```
RunTask(task)
    ├─ Validate task
    ├─ Resolve inputs (InputResolver)
    ├─ Compute hash (TaskHasher)
    ├─ Check cache
    │   └─ HIT: Replay(entry) → return RunResult{FromCache: true}
    ├─ MISS: Execute(task, hash)
    ├─ If ExitCode == 0: Harvest(outputs)
    ├─ Cache.Put(entry)
    └─ return RunResult{FromCache: false}
```

### Prohibited Behaviors (Verified NOT Implemented)

Per spec.md:
- ❌ No partial artifact capture on failure
- ❌ No re-execution on cache hit (even for failures)
- ❌ No autodiscovery of changed files

### Total Tests: 89 passing