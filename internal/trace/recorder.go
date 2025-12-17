package trace

import "sync"

// Sink is the minimal interface the execution engine depends on.
//
// Record must be inert:
//   - must not panic (implementations should guard themselves)
//   - must not return errors
//
// The caller must assume Record may be a no-op.
type Sink interface {
	Record(event TraceEvent)
}

// NopSink discards all events.
type NopSink struct{}

func (NopSink) Record(TraceEvent) {}

// SafeRecord records an event and guarantees inertness even if the sink is buggy.
// It intentionally swallows panics.
func SafeRecord(s Sink, event TraceEvent) {
	if s == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	s.Record(event)
}

// Recorder is a concurrency-safe in-memory collector.
//
// Concurrency note:
// Recording uses a single mutex. This may add contention, but it does not affect
// the canonical trace ordering because ordering is computed after collection.
//
// Safety note:
// Record never panics (it recovers internally) and never returns an error.
type Recorder struct {
	mu     sync.Mutex
	events []TraceEvent
}

func NewRecorder() *Recorder { return &Recorder{} }

func (r *Recorder) Record(event TraceEvent) {
	if r == nil {
		return
	}
	defer func() {
		_ = recover()
	}()

	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()
}

// Snapshot returns a point-in-time copy of all recorded events.
func (r *Recorder) Snapshot() []TraceEvent {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]TraceEvent, len(r.events))
	copy(out, r.events)
	return out
}

// Trace builds an ExecutionTrace from the currently recorded events.
// The returned trace is independent from the recorder (events are copied).
func (r *Recorder) Trace(graphHash string) ExecutionTrace {
	tr := ExecutionTrace{GraphHash: graphHash}
	tr.Events = r.Snapshot()
	tr.Canonicalize()
	return tr
}
