package trace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// ExecutionTrace is the canonical, deterministic record of a graph execution.
//
// Invariants (from sprint-03 trace-engine spec):
//   - Must capture GraphHash and an ordered list of events.
//   - Must contain logical transitions/decisions, not runtime-dependent details.
//   - Must not include timestamps, pointers, or any runtime-dependent values.
//
// Note: GraphHash is a string to avoid coupling this package to a specific graph implementation.
// It should be populated with the deterministic graph identity.
//
// Canonical representation:
//   - Events are sorted via Canonicalize() using a fully-specified ordering.
//   - JSON serialization uses a custom marshaler to fix field order and omit absent optional fields.
//
// Any consumer producing traces should treat ExecutionTrace as immutable once Canonicalize() is called.
// The trace is observational only and must never affect execution behavior.
//
// This model is intentionally minimal: it captures only the logical facts required by the sprint-03 spec.
// Future prompts may expand event payloads.
//
// IMPORTANT: This is the source of truth for "what happened"; byte-for-byte stability is required.
//
// See docs/sprints/sprint-03/in-process/trace-engine/spec.md for rationale and constraints.
type ExecutionTrace struct {
	GraphHash string
	Events    []TraceEvent
}

// TraceEventKind is the stable, canonical discriminator for TraceEvent.
//
// These kinds represent logical decisions/transitions, not runtime occurrences.
// The string values are part of the trace's canonical bytes; do not rename.
type TraceEventKind string

const (
	EventTaskInvalidated      TraceEventKind = "TaskInvalidated"
	EventTaskArtifactsRestored TraceEventKind = "TaskArtifactsRestored"
	EventTaskCached           TraceEventKind = "TaskCached"
	EventTaskExecuted         TraceEventKind = "TaskExecuted"
	EventTaskFailed           TraceEventKind = "TaskFailed"
	EventTaskSkipped          TraceEventKind = "TaskSkipped"
)

// TraceEvent is a single logical transition/decision.
//
// Determinism constraints:
//   - No timestamps.
//   - No error strings / stack traces.
//   - No fields derived from pointer identity or map iteration.
//
// Optional fields must be set deterministically and canonicalized:
//   - Empty slices are normalized to nil (omitted in JSON).
//   - Artifacts are sorted.
type TraceEvent struct {
	Kind TraceEventKind

	// TaskID identifies the task/node this event refers to. For task-level events this is required.
	TaskID string

	// Reason is a stable, logical reason code (e.g., "InputChanged", "UpstreamFailed").
	// The set of allowed values is intentionally open in this sprint; producers must keep them stable.
	Reason string

	// CauseTaskID records a related upstream task (e.g., the failing upstream task causing a skip).
	CauseTaskID string

	// Artifacts is a list of restored artifact identifiers. The producer must ensure identifiers are stable.
	Artifacts []string
}

// Validate checks basic invariants and returns a descriptive error.
func (t *ExecutionTrace) Validate() error {
	if t == nil {
		return errors.New("trace is nil")
	}
	if t.GraphHash == "" {
		return errors.New("graphHash is required")
	}
	for i := range t.Events {
		e := t.Events[i]
		if e.Kind == "" {
			return fmt.Errorf("events[%d].kind is required", i)
		}
		if isTaskEvent(e.Kind) && e.TaskID == "" {
			return fmt.Errorf("events[%d].taskId is required for kind %q", i, e.Kind)
		}
		if len(e.Artifacts) > 0 {
			for j, a := range e.Artifacts {
				if a == "" {
					return fmt.Errorf("events[%d].artifacts[%d] is empty", i, j)
				}
			}
		}
	}
	return nil
}

func isTaskEvent(kind TraceEventKind) bool {
	switch kind {
	case EventTaskInvalidated, EventTaskArtifactsRestored, EventTaskCached, EventTaskExecuted, EventTaskFailed, EventTaskSkipped:
		return true
	default:
		return true
	}
}

// Canonicalize normalizes and sorts the trace into its canonical form.
//
// Ordering guarantee (from spec): ordering is independent of execution timing or concurrency.
// This implementation produces a total order over events, with TaskID as the primary key.
//
// Canonicalization rules:
//   - Artifacts are copied and sorted.
//   - Empty Artifacts slices are normalized to nil.
//   - Events are stably sorted by (taskId, kindOrder, reason, causeTaskId, artifactsLex).
func (t *ExecutionTrace) Canonicalize() {
	if t == nil {
		return
	}
	for i := range t.Events {
		if len(t.Events[i].Artifacts) == 0 {
			t.Events[i].Artifacts = nil
			continue
		}
		art := make([]string, len(t.Events[i].Artifacts))
		copy(art, t.Events[i].Artifacts)
		sort.Strings(art)
		t.Events[i].Artifacts = art
	}

	sort.SliceStable(t.Events, func(i, j int) bool {
		a := t.Events[i]
		b := t.Events[j]

		if a.TaskID != b.TaskID {
			return a.TaskID < b.TaskID
		}
		if kindOrder(a.Kind) != kindOrder(b.Kind) {
			return kindOrder(a.Kind) < kindOrder(b.Kind)
		}
		if a.Reason != b.Reason {
			return a.Reason < b.Reason
		}
		if a.CauseTaskID != b.CauseTaskID {
			return a.CauseTaskID < b.CauseTaskID
		}
		return compareStringSlices(a.Artifacts, b.Artifacts)
	})
}

func kindOrder(k TraceEventKind) int {
	switch k {
	case EventTaskInvalidated:
		return 10
	case EventTaskArtifactsRestored:
		return 20
	case EventTaskCached:
		return 30
	case EventTaskExecuted:
		return 40
	case EventTaskFailed:
		return 50
	case EventTaskSkipped:
		return 60
	default:
		return 1000
	}
}

func compareStringSlices(a, b []string) bool {
	// nil and empty are treated identically by Canonicalize (empties are normalized to nil).
	la := len(a)
	lb := len(b)
	min := la
	if lb < min {
		min = lb
	}
	for i := 0; i < min; i++ {
		if a[i] == b[i] {
			continue
		}
		return a[i] < b[i]
	}
	return la < lb
}

// CanonicalJSON returns the canonical JSON encoding of the trace.
// It canonicalizes a copy of the trace to avoid mutating the caller's slices.
func (t ExecutionTrace) CanonicalJSON() ([]byte, error) {
	copyTrace := ExecutionTrace{GraphHash: t.GraphHash}
	copyTrace.Events = make([]TraceEvent, len(t.Events))
	copy(copyTrace.Events, t.Events)
	copyTrace.Canonicalize()
	if err := copyTrace.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&copyTrace)
}

// Hash returns the deterministic trace hash (sha256 hex) of the canonical JSON bytes.
func (t ExecutionTrace) Hash() (string, error) {
	b, err := t.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return ComputeTraceHash(b), nil
}

// MarshalJSON ensures canonical field ordering and omission rules.
func (t ExecutionTrace) MarshalJSON() ([]byte, error) {
	// Canonicalization is the responsibility of CanonicalJSON(), but MarshalJSON should still be stable.
	// We do not sort here to avoid surprising mutation; field ordering is deterministic regardless.
	if t.GraphHash == "" {
		return nil, errors.New("graphHash is required")
	}
	var buf bytes.Buffer
	buf.WriteByte('{')

	// graphHash
	buf.WriteString("\"graphHash\":")
	gh, _ := json.Marshal(t.GraphHash)
	buf.Write(gh)
	buf.WriteByte(',')

	// events
	buf.WriteString("\"events\":[")
	for i := range t.Events {
		if i > 0 {
			buf.WriteByte(',')
		}
		eb, err := json.Marshal(t.Events[i])
		if err != nil {
			return nil, err
		}
		buf.Write(eb)
	}
	buf.WriteByte(']')

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// MarshalJSON ensures canonical field ordering and omission of empty optional fields.
func (e TraceEvent) MarshalJSON() ([]byte, error) {
	if e.Kind == "" {
		return nil, errors.New("kind is required")
	}
	// Canonicalize per-event slice normalization without mutating the original slice.
	var artifacts []string
	if len(e.Artifacts) > 0 {
		artifacts = make([]string, len(e.Artifacts))
		copy(artifacts, e.Artifacts)
		sort.Strings(artifacts)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')

	// kind (always first)
	buf.WriteString("\"kind\":")
	kb, _ := json.Marshal(string(e.Kind))
	buf.Write(kb)

	// taskId
	if e.TaskID != "" {
		buf.WriteByte(',')
		buf.WriteString("\"taskId\":")
		tb, _ := json.Marshal(e.TaskID)
		buf.Write(tb)
	}

	// reason
	if e.Reason != "" {
		buf.WriteByte(',')
		buf.WriteString("\"reason\":")
		rb, _ := json.Marshal(e.Reason)
		buf.Write(rb)
	}

	// causeTaskId
	if e.CauseTaskID != "" {
		buf.WriteByte(',')
		buf.WriteString("\"causeTaskId\":")
		cb, _ := json.Marshal(e.CauseTaskID)
		buf.Write(cb)
	}

	// artifacts
	if len(artifacts) > 0 {
		buf.WriteByte(',')
		buf.WriteString("\"artifacts\":[")
		for i := range artifacts {
			if i > 0 {
				buf.WriteByte(',')
			}
			ab, _ := json.Marshal(artifacts[i])
			buf.Write(ab)
		}
		buf.WriteByte(']')
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}
