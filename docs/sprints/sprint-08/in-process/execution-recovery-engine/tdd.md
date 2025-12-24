# Sprint-08 Test Strategy

## Unit Tests

* Failure classification correctness
* Checkpoint validation logic
* Resume eligibility computation
* Retry counter enforcement

## Integration Tests

### 1. Happy Resume

* Execute nodes A &rarr; B &rarr; C
* Fail at D
* Resume continues from D

### 2. Upstream Invalidation

* Change input of node B
* Resume invalidates C and D
* Execution restarts from B

### 3. Non-Resumable Failure

* Graph schema error
* Resume attempt rejected

### 4. Workspace Corruption

* Manual deletion of cache entry
* Resume rejected

### 5. Crash Recovery

* Simulated crash mid-execution
* Resume from last valid checkpoint

### 6. Run Linking
* Resume maps to correct `previous_run_id`
* Resume rejected if `previous_run_id` does not exist

## Deterministic Tests

* Same failure &rarr; same resume point
* Same retry &rarr; same outputs
* Different graph hash &rarr; full reset
