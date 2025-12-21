# Test-Driven Development Plan &mdash; Graph Schema v1

## Testing Philosophy

Tests define the contract. Implementation must conform to tests, not vice versa.

## Test Categories

### 1. Schema Validation Tests

* Valid minimal graph passes
* Missing required fields fail
* Unknown fields fail
* Incorrect types fail

### 2. Structural Validation Tests

* Cyclic graph rejected
* Duplicate node IDs rejected
* Dangling edges rejected
* Self-referential edges rejected

### 3. Normalization Tests

* Field order differences normalize identically
* Default values injected deterministically
* Equivalent graphs normalize byte-identically

### 4. Hash Stability Tests

* Same graph &rarr; same hash across runs
* Reordered JSON &rarr; same hash
* Whitespace-only changes &rarr; same hash
* Metadata changes &rarr; same hash

### 5. Hash Sensitivity Tests

* Semantic change &rarr; different hash
* Node input change &rarr; different hash
* Edge change &rarr; different hash

### 6. Versioning Tests

* Missing `schema_version` fails
* Unsupported version fails
* Supported version parses

### 7. Golden Fixtures

* minimal.graph.json
* maximal.graph.json
* invalid.graph.json
* cyclic.graph.json

Each fixture has:

* Expected validation result
* Expected hash (if valid)

## Failure Taxonomy Tests

Errors must be categorized as:

* ParseError
* SchemaError
* StructuralError
* SemanticError

Error messages must be deterministic.

## Regression Policy

* Hash outputs are locked
* Any change requires explicit version bump

## CI Requirements

* All tests must run without environment dependencies
* No nondeterministic inputs allowed
