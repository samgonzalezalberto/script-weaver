package trace

import (
	"bytes"
	"testing"
)

func TestCanonicalTraceStability_ByteForByte(t *testing.T) {
	trace1 := ExecutionTrace{
		GraphHash: "graph-abc",
		Events: []TraceEvent{
			{Kind: EventTaskExecuted, TaskID: "b"},
			{Kind: EventTaskCached, TaskID: "a"},
			{Kind: EventTaskSkipped, TaskID: "c", Reason: "UpstreamFailed", CauseTaskID: "b"},
		},
	}

	trace2 := ExecutionTrace{
		GraphHash: "graph-abc",
		Events: []TraceEvent{
			{Kind: EventTaskSkipped, TaskID: "c", CauseTaskID: "b", Reason: "UpstreamFailed"},
			{Kind: EventTaskCached, TaskID: "a"},
			{Kind: EventTaskExecuted, TaskID: "b"},
		},
	}

	b1, err := trace1.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical json (1): %v", err)
	}
	b2, err := trace2.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical json (2): %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Fatalf("expected identical bytes\n1=%s\n2=%s", string(b1), string(b2))
	}
}

func TestCanonicalOrdering_SortsByTaskID(t *testing.T) {
	tr := ExecutionTrace{
		GraphHash: "graph-abc",
		Events: []TraceEvent{
			{Kind: EventTaskExecuted, TaskID: "b"},
			{Kind: EventTaskExecuted, TaskID: "a"},
		},
	}
	b, err := tr.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}
	// Expect task a before b.
	expected := `{"graphHash":"graph-abc","events":[{"kind":"TaskExecuted","taskId":"a"},{"kind":"TaskExecuted","taskId":"b"}]}`
	if string(b) != expected {
		t.Fatalf("unexpected canonical bytes\nexpected=%s\nactual  =%s", expected, string(b))
	}
}

func TestHash_Deterministic(t *testing.T) {
	tr1 := ExecutionTrace{GraphHash: "g", Events: []TraceEvent{{Kind: EventTaskCached, TaskID: "a"}}}
	tr2 := ExecutionTrace{GraphHash: "g", Events: []TraceEvent{{Kind: EventTaskCached, TaskID: "a"}}}

	h1, err := tr1.Hash()
	if err != nil {
		t.Fatalf("hash (1): %v", err)
	}
	h2, err := tr2.Hash()
	if err != nil {
		t.Fatalf("hash (2): %v", err)
	}
	if h1 != h2 {
		t.Fatalf("expected identical hash, got %q != %q", h1, h2)
	}
}

func TestHash_IgnoresInsertionOrder_WhenSemanticallyEquivalent(t *testing.T) {
	tr1 := ExecutionTrace{
		GraphHash: "g",
		Events: []TraceEvent{
			{Kind: EventTaskExecuted, TaskID: "b", Reason: "FreshWork"},
			{Kind: EventTaskCached, TaskID: "a", Reason: "CacheHit"},
		},
	}
	tr2 := ExecutionTrace{
		GraphHash: "g",
		Events: []TraceEvent{
			{Kind: EventTaskCached, TaskID: "a", Reason: "CacheHit"},
			{Kind: EventTaskExecuted, TaskID: "b", Reason: "FreshWork"},
		},
	}

	h1, err := tr1.Hash()
	if err != nil {
		t.Fatalf("hash (1): %v", err)
	}
	h2, err := tr2.Hash()
	if err != nil {
		t.Fatalf("hash (2): %v", err)
	}
	if h1 != h2 {
		t.Fatalf("expected equal hash for semantically equivalent traces, got %q != %q", h1, h2)
	}
}

func TestEventArtifacts_CanonicalizedAndOmittedWhenEmpty(t *testing.T) {
	tr := ExecutionTrace{
		GraphHash: "g",
		Events: []TraceEvent{{
			Kind:      EventTaskArtifactsRestored,
			TaskID:    "a",
			Artifacts: []string{"z", "a"},
		}},
	}
	b, err := tr.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}
	expected := `{"graphHash":"g","events":[{"kind":"TaskArtifactsRestored","taskId":"a","artifacts":["a","z"]}]}`
	if string(b) != expected {
		t.Fatalf("unexpected canonical bytes\nexpected=%s\nactual  =%s", expected, string(b))
	}

	tr2 := ExecutionTrace{GraphHash: "g", Events: []TraceEvent{{Kind: EventTaskCached, TaskID: "a", Artifacts: []string{}}}}
	b2, err := tr2.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}
	expected2 := `{"graphHash":"g","events":[{"kind":"TaskCached","taskId":"a"}]}`
	if string(b2) != expected2 {
		t.Fatalf("unexpected canonical bytes\nexpected=%s\nactual  =%s", expected2, string(b2))
	}
}
