# Notes — Phase 1: Error Taxonomy

## Error Handling Strategy

**Decision**: Use custom struct types **with** sentinel errors.

### Why This Approach?

1. **Programmatic Checking**: Sentinel errors (`ErrParse`, `ErrSchema`, etc.) enable `errors.Is()` checks
2. **Rich Context**: Struct types carry additional fields (`Field`, `Msg`, `Kind`) for deterministic error messages
3. **Pattern Consistency**: Follows existing `internal/dag/errors.go` pattern using `Unwrap()` method
4. **Test Compatibility**: Tests can use `errors.Is(err, graph.ErrSchema)` to categorize failures

### Error Type Summary

| Type | Sentinel | Use Case |
|------|----------|----------|
| `ParseError` | `ErrParse` | JSON decode failures, encoding issues |
| `SchemaError` | `ErrSchema` | Missing fields, wrong types, unknown fields |
| `StructuralError` | `ErrStructural` | Cycles, duplicate IDs, dangling edges |
| `SemanticError` | `ErrSemantic` | Invalid version, logic violations |

### Determinism

All `Error()` methods produce deterministic output by:
- Using fixed format strings
- Not including timestamps, random values, or pointer addresses
- Sorting any list elements before formatting (to be implemented in constructors)

## Assumptions

1. Error messages must be deterministic (per TDD spec)
2. Errors must be programmatically distinguishable (for test assertions)
3. No external dependencies required for error types

---

# Notes — Phase 2: Schema Implementation & Parsing

## Ambiguities Resolved

### 1. `metadata` Object — Required but Empty
- **Spec says**: `metadata` is a required top-level field
- **Interpretation**: An empty `{}` satisfies the requirement
- **Decision**: Accept empty metadata objects, no sub-fields are required

### 2. `inputs` Type — `map[string]any`
- **Spec says**: "Immutable `inputs` (key-value map)"
- **Data Dictionary says**: "Type: object (map<string, any>)"
- **Decision**: Use `map[string]any` to allow arbitrary input values
- **Ambiguity**: Spec doesn't define valid value types — accepting `any` for now

### 3. Unknown Field Errors — ParseError vs SchemaError
- **Issue**: `DisallowUnknownFields` returns error during JSON decode
- **Decision**: Classify as `ParseError` (happens at parse time)
- **Alternative**: Could wrap with `SchemaError` — chose simpler approach

### 4. Struct Zero Values vs Missing Fields
- **Issue**: Go structs have zero values even for missing JSON fields
- **Decision**: For required string fields, check `== ""` to detect missing
- **For slices/maps**: Check `== nil` — explicit `[]` in JSON yields non-nil

## Error Classification

| Condition | Error Type |
|-----------|------------|
| Malformed JSON | `ParseError` |
| Unknown field | `ParseError` |
| Wrong type | `SchemaError` |
| Missing required field | `SchemaError` |
| Unsupported version | `SemanticError` |

---

# Notes — Phase 3: Structural Validation

## Cycle Detection Algorithm

**Choice**: DFS with three-color marking (white/gray/black)

### Why DFS with Coloring?

1. **O(V+E) complexity**: Linear time for graph traversal
2. **Clear semantics**: Gray = in-progress (back-edge to gray = cycle)
3. **Path reconstruction**: Gray nodes form the current path for error messages
4. **Determinism**: Sorting nodes/edges before traversal ensures reproducible errors

### Algorithm

```
white (0) = unvisited
gray  (1) = in progress (on current path)
black (2) = finished
```

If DFS encounters a gray node, a cycle exists.

### Validation Order

1. **Duplicate IDs** — Sort nodes by ID, report first duplicate
2. **Self-reference** — Check before building adjacency
3. **Dangling edges** — Check during edge iteration
4. **Cycles** — DFS on sorted node list

All checks use sorted iteration for **deterministic error reporting**.

---

# Notes — Phase 4: Canonical Normalization

## Sorting Strategy

- **Nodes**: Sorted by `id` using Go's `<` operator (byte-wise comparison)
- **Edges**: Sorted by `from`, then `to` (lexicographic tuple sort)
- **Outputs**: Sorted using `sort.Strings()` (byte-wise)
- **Inputs**: Keys sorted by `encoding/json` on marshal (guaranteed by Go spec)

## Unicode Edge Cases

> [!NOTE]
> Go's string comparison uses **byte-wise** ordering, not Unicode collation.

**Implications:**
- `"a" < "b"` ✅ (as expected)
- `"Z" < "a"` ✅ (uppercase before lowercase in ASCII)
- `"é" > "z"` — Extended Unicode characters sort after ASCII

**Decision**: Use byte-wise sorting (Go default) for simplicity.
- Matches `encoding/json` key sorting behavior
- Deterministic across all Go implementations
- If locale-aware sorting is ever needed, it would require explicit collation

## Determinism Guarantees

1. `Normalize()` + `json.Marshal()` produces **identical bytes** for semantically equivalent graphs
2. Map key ordering is handled by `encoding/json` (sorts keys alphabetically)
3. Empty slices `[]` and `nil` slices both normalize to `[]` in JSON

---

# Notes — Phase 5: Hash Stability

## Hash Algorithm

**SHA-256** with hex encoding (64 character output)

## Serialization Format

The hash is computed from:

```
json.Marshal(graph.Normalized())
```

**Included in hash:**
- `nodes[]` — id, type, inputs, outputs
- `edges[]` — from, to

**Excluded from hash:**
- `schema_version` (top-level)
- `metadata` (top-level)

## Canonical JSON Format

```json
{"edges":[{"from":"a","to":"b"}],"nodes":[{"id":"a","inputs":{},"outputs":[],"type":"t"}]}
```

Properties:
- Compact (no whitespace)
- Keys sorted alphabetically (handled by `encoding/json`)
- Arrays sorted by `Normalize()` rules

## Stability Contract

> [!IMPORTANT]
> v1 hashes are locked forever. Any change to normalization or serialization requires a version bump.

---

# Notes — Phase 6: Regression Locking

## Golden Fixture Hashes

> [!CAUTION]
> These hashes are **locked** as the v1 contract. Changes require a version bump.

| Fixture | File | Locked Hash (SHA-256) |
|---------|------|----------------------|
| Minimal | `minimal.graph.json` | `a461bf77bc4e4d732f7afc121c70e7f70ed8bf225a082a4e01951d1eb6b5c278` |
| Maximal | `maximal.graph.json` | `87f41d22ad26e2102bfd37bf69cc45866886873cb2292563efe800ab5f92fc9a` |

## Invalid Fixtures

| Fixture | Expected Error |
|---------|----------------|
| `cyclic.graph.json` | `StructuralError` (kind: `cycle`) |
| `duplicate_id.graph.json` | `StructuralError` (kind: `duplicate_id`) |

## Hash Verification

Run to verify locked hashes:
```bash
go test ./internal/graph/... -run TestGolden -v
```
