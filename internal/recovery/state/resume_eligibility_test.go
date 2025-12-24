package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"scriptweaver/internal/incremental"
)

func TestResumeEligibilityChecker_Allows_WhenRulesSatisfied(t *testing.T) {
	root := t.TempDir()
	store, _ := NewStore(root)

	prevRun := Run{
		RunID:         "prev",
		GraphHash:     "gh",
		StartTime:     time.Unix(1, 0).UTC(),
		Mode:          ExecutionModeIncremental,
		RetryCount:    0,
		Status:        "failed",
		PreviousRunID: nil,
	}
	if err := store.SaveRun(prevRun); err != nil {
		t.Fatalf("SaveRun(prev): %v", err)
	}
	if err := store.SaveFailure("prev", Failure{FailureClass: FailureClassSystem, ErrorCode: "CRASH", ErrorMessage: "crash", Resumable: true}); err != nil {
		t.Fatalf("SaveFailure(prev): %v", err)
	}

	prevID := "prev"
	newRun := Run{
		RunID:         "new",
		GraphHash:     "gh",
		StartTime:     time.Unix(2, 0).UTC(),
		Mode:          ExecutionModeIncremental,
		RetryCount:    1,
		Status:        "running",
		PreviousRunID: &prevID,
	}

	g := &incremental.GraphSnapshot{Nodes: map[string]incremental.NodeSnapshot{
		"A": {Name: "A", Upstream: []string{}},
		"B": {Name: "B", Upstream: []string{"A"}},
	}}
	inv := incremental.InvalidationMap{
		"A": {Invalidated: false},
		"B": {Invalidated: false},
	}

	checker := &ResumeEligibilityChecker{Store: store, ProjectRoot: root}
	if err := checker.Check(ResumeEligibilityRequest{NewRun: newRun, ResumeFromNodeID: "B", Graph: g, Invalidation: inv}); err != nil {
		t.Fatalf("expected eligible, got error: %v", err)
	}
}

func TestResumeEligibilityChecker_Rejects_WhenGraphHashDiffers(t *testing.T) {
	root := t.TempDir()
	store, _ := NewStore(root)

	prev := Run{RunID: "prev", GraphHash: "gh1", StartTime: time.Unix(1, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 0, Status: "failed"}
	_ = store.SaveRun(prev)
	_ = store.SaveFailure("prev", Failure{FailureClass: FailureClassSystem, ErrorCode: "CRASH", ErrorMessage: "crash", Resumable: true})

	prevID := "prev"
	newRun := Run{RunID: "new", GraphHash: "gh2", StartTime: time.Unix(2, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 1, Status: "running", PreviousRunID: &prevID}

	checker := &ResumeEligibilityChecker{Store: store, ProjectRoot: root}
	err := checker.Check(ResumeEligibilityRequest{NewRun: newRun, ResumeFromNodeID: "A", Graph: &incremental.GraphSnapshot{Nodes: map[string]incremental.NodeSnapshot{"A": {Name: "A"}}}, Invalidation: incremental.InvalidationMap{"A": {Invalidated: false}}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResumeEligibilityChecker_Rejects_WhenUpstreamInvalidated(t *testing.T) {
	root := t.TempDir()
	store, _ := NewStore(root)

	prev := Run{RunID: "prev", GraphHash: "gh", StartTime: time.Unix(1, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 0, Status: "failed"}
	_ = store.SaveRun(prev)
	_ = store.SaveFailure("prev", Failure{FailureClass: FailureClassExecution, ErrorCode: "E", ErrorMessage: "err", Resumable: true})

	prevID := "prev"
	newRun := Run{RunID: "new", GraphHash: "gh", StartTime: time.Unix(2, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 1, Status: "running", PreviousRunID: &prevID}

	g := &incremental.GraphSnapshot{Nodes: map[string]incremental.NodeSnapshot{
		"A": {Name: "A", Upstream: []string{}},
		"B": {Name: "B", Upstream: []string{"A"}},
		"C": {Name: "C", Upstream: []string{"B"}},
	}}
	inv := incremental.InvalidationMap{
		"A": {Invalidated: true},
		"B": {Invalidated: false},
		"C": {Invalidated: false},
	}

	checker := &ResumeEligibilityChecker{Store: store, ProjectRoot: root}
	if err := checker.Check(ResumeEligibilityRequest{NewRun: newRun, ResumeFromNodeID: "C", Graph: g, Invalidation: inv}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestResumeEligibilityChecker_Rejects_WhenRetryCountNotIncremented(t *testing.T) {
	root := t.TempDir()
	store, _ := NewStore(root)

	prev := Run{RunID: "prev", GraphHash: "gh", StartTime: time.Unix(1, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 5, Status: "failed"}
	_ = store.SaveRun(prev)
	_ = store.SaveFailure("prev", Failure{FailureClass: FailureClassSystem, ErrorCode: "CRASH", ErrorMessage: "crash", Resumable: true})

	prevID := "prev"
	newRun := Run{RunID: "new", GraphHash: "gh", StartTime: time.Unix(2, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 5, Status: "running", PreviousRunID: &prevID}

	checker := &ResumeEligibilityChecker{Store: store, ProjectRoot: root}
	err := checker.Check(ResumeEligibilityRequest{NewRun: newRun, ResumeFromNodeID: "A", Graph: &incremental.GraphSnapshot{Nodes: map[string]incremental.NodeSnapshot{"A": {Name: "A"}}}, Invalidation: incremental.InvalidationMap{"A": {Invalidated: false}}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResumeEligibilityChecker_Rejects_WhenWorkspaceUnauthorizedEntry(t *testing.T) {
	root := t.TempDir()
	store, _ := NewStore(root)

	// Create unauthorized file under .scriptweaver
	if err := os.MkdirAll(filepath.Join(root, ".scriptweaver"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".scriptweaver", "hax.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	prev := Run{RunID: "prev", GraphHash: "gh", StartTime: time.Unix(1, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 0, Status: "failed"}
	_ = store.SaveRun(prev)
	_ = store.SaveFailure("prev", Failure{FailureClass: FailureClassSystem, ErrorCode: "CRASH", ErrorMessage: "crash", Resumable: true})

	prevID := "prev"
	newRun := Run{RunID: "new", GraphHash: "gh", StartTime: time.Unix(2, 0).UTC(), Mode: ExecutionModeIncremental, RetryCount: 1, Status: "running", PreviousRunID: &prevID}

	checker := &ResumeEligibilityChecker{Store: store, ProjectRoot: root}
	err := checker.Check(ResumeEligibilityRequest{NewRun: newRun, ResumeFromNodeID: "A", Graph: &incremental.GraphSnapshot{Nodes: map[string]incremental.NodeSnapshot{"A": {Name: "A"}}}, Invalidation: incremental.InvalidationMap{"A": {Invalidated: false}}})
	if err == nil {
		t.Fatalf("expected error")
	}
}
