// Package core provides the domain models for deterministic task execution.
//
// These models are derived directly from the frozen specifications:
//   - docs/sprints/sprint-00/planning/spec.md
//   - docs/sprints/sprint-00/planning/data-dictionary.md
//
// # Design Principles
//
// All structures in this package adhere to the following constraints:
//
//  1. No implied fields that could affect determinism (e.g., timestamps)
//  2. All fields correspond to explicit specification requirements
//  3. Structures support exact serialization for reproducible hashing
//
// # Core Types
//
// Task: A declarative definition of work to be executed deterministically.
// Input: A resolved file whose content contributes to task identity.
// Artifact: A file produced by a task and declared in outputs.
//
// See the data-dictionary.md for canonical definitions of each term.
package core
