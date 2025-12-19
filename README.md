# ScriptWeaver

A deterministic task execution tool written in Go. ScriptWeaver ensures that identical inputs always produce identical outputs, enabling safe caching and replay of task results.

## Features

- **Deterministic Execution** — Tasks execute in isolated, controlled environments with explicit inputs and outputs
- **Content-Based Caching** — Results are cached based on file contents, not timestamps
- **Reproducible Builds** — Identical inputs always produce identical outputs across runs and machines
- **DAG Support** — Tasks can be organized as directed acyclic graphs with automatic dependency resolution
- **Incremental Builds** — Only re-execute tasks when their inputs change

## Installation

### Prerequisites

- Go 1.22 or later

### From Source

```bash
git clone https://github.com/samgonzalez27/script-weaver.git
cd script-weaver
go build -o scriptweaver ./cmd/scriptweaver
```

## Quick Start

```bash
# Run scriptweaver
./scriptweaver [options]
```

## How It Works

ScriptWeaver computes a **Task Hash** for each task based on:

- Sorted list of input file contents
- Expanded and sorted input paths
- Task command (`run`)
- Explicit environment variables (`env`)
- Declared outputs
- Working directory identity

If a Task Hash matches a previous execution, cached results are replayed exactly—including stdout, stderr, and exit code.

## Task Definition

Tasks are defined declaratively using structured configuration (YAML or JSON):

```yaml
name: build
inputs:
  - src/**/*.go
  - go.mod
  - go.sum
run: go build -o ./bin/app ./cmd/app
outputs:
  - bin/app
env:
  CGO_ENABLED: "0"
```

### Required Fields

| Field    | Description                                      |
|----------|--------------------------------------------------|
| `name`   | Logical identifier for the task                  |
| `inputs` | List of file paths or glob patterns              |
| `run`    | Command string to execute                        |

### Optional Fields

| Field     | Description                                                |
|-----------|------------------------------------------------------------|
| `env`     | Map of environment variables (only these are visible)      |
| `outputs` | List of file paths/directories produced by the task        |

## Deterministic Guarantees

1. **Input Determinism** — Glob expansion is strictly sorted; file ordering is stable
2. **Environment Determinism** — Only declared environment variables are visible
3. **Execution Determinism** — Tasks run in isolated environments
4. **Output Determinism** — Outputs are normalized to remove non-deterministic data

## Project Structure

```
script-weaver/
├── cmd/scriptweaver/     # CLI entrypoint
├── cli/                  # CLI tests
├── internal/
│   ├── cli/              # CLI parsing and execution
│   ├── core/             # Domain models (Task, Input, Artifact)
│   ├── dag/              # DAG construction and traversal
│   ├── incremental/      # Incremental build support
│   └── trace/            # Execution tracing
├── docs/sprints/         # Sprint planning and documentation
└── go.mod
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o scriptweaver ./cmd/scriptweaver
```

## Documentation

Detailed specifications and design documents are available in the `docs/sprints/` directory, organized by development sprint.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

