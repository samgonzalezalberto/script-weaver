package state

import "testing"

func TestFailureFromError_ClassifiesGraphFailure(t *testing.T) {
	f, err := failureFromError(&GraphFailureError{Code: "SchemaViolation", Message: "bad"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.FailureClass != FailureClassGraph || f.Resumable || f.NodeID != nil {
		t.Fatalf("unexpected failure: %#v", f)
	}
}

func TestFailureFromError_ClassifiesWorkspaceFailure(t *testing.T) {
	f, err := failureFromError(&WorkspaceFailureError{Code: "WorkspaceInvalid", Message: "bad"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.FailureClass != FailureClassWorkspace || f.Resumable || f.NodeID != nil {
		t.Fatalf("unexpected failure: %#v", f)
	}
}

func TestFailureFromError_ClassifiesExecutionFailure(t *testing.T) {
	f, err := failureFromError(&ExecutionFailureError{NodeID: "A", Code: "NodeFailed", Message: "bad"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.FailureClass != FailureClassExecution || !f.Resumable || f.NodeID == nil || *f.NodeID != "A" {
		t.Fatalf("unexpected failure: %#v", f)
	}
}

func TestFailureFromError_ClassifiesSystemFailure(t *testing.T) {
	f, err := failureFromError(&SystemFailureError{Code: "Panic", Message: "boom"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.FailureClass != FailureClassSystem || !f.Resumable || f.NodeID != nil {
		t.Fatalf("unexpected failure: %#v", f)
	}
}
