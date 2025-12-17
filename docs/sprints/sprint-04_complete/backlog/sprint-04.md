# Sprint-04 Backlog

## Deferred Capabilities

### 1. Granular Reason Types: DeclaredInputsChanged
*   **Description**: Introduce a dedicated `DeclaredInputsChanged` reason type.
*   **Context**: Sprint-04 mapped declared input definition changes to `GraphStructureChanged` with specific details (`DeclaredInputs=changed`) to simplify the type system. Separating definition changes from content changes may improve clarity in future sprints.
*   **Dependency**: Sprint-04 Invalidation Engine.

### 2. Cross-Machine Invalidation Distribution
*   **Description**: Transport and application of invalidation maps across networked machines.
*   **Context**: Explicitly out of scope for Sprint-04. While the current implementation guarantees machine-independent *determinism* (canonical serialization), the actual network transport and synchronization protocols are not implemented.
*   **Dependency**: Distributed Execution / Remote Cache.

### 3. Usage-Based Invalidation Optimization
*   **Description**: Optimization of invalidation payload size for extremely large graphs.
*   **Context**: The current implementation stores full, sorted details for all invalidations. Efficiency improvements (interning, compression) were deferred to prioritize correctness and determinism.
*   **Dependency**: Performance Profiling of Sprint-04.

### 4. Plugin Extension Points
*   **Description**: APIs for third-party plugins to inject custom invalidation reasons.
*   **Context**: Explicitly excluded to ensure the core invalidation logic remains hermetic and verifiable during the foundational phase.
*   **Dependency**: Plugin System Spec.
