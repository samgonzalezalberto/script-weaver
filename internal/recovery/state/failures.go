package state

import (
	"errors"
	"fmt"
)

// GraphFailureError represents deterministic graph validation failures.
// Not resumable (per frozen spec).
type GraphFailureError struct {
	Code    string
	Message string
	Cause   error
}

func (e *GraphFailureError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("graph failure (%s): %s", e.Code, e.Message)
	}
	return fmt.Sprintf("graph failure: %s", e.Message)
}

func (e *GraphFailureError) Unwrap() error { return e.Cause }

// WorkspaceFailureError represents workspace corruption or invalid workspace structure.
// Not resumable (per frozen spec).
type WorkspaceFailureError struct {
	Code    string
	Message string
	Cause   error
}

func (e *WorkspaceFailureError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("workspace failure (%s): %s", e.Code, e.Message)
	}
	return fmt.Sprintf("workspace failure: %s", e.Message)
}

func (e *WorkspaceFailureError) Unwrap() error { return e.Cause }

// ExecutionFailureError represents a node-level execution failure.
// Conditionally resumable (per frozen spec).
type ExecutionFailureError struct {
	NodeID  string
	Code    string
	Message string
	Cause   error
}

func (e *ExecutionFailureError) Error() string {
	if e == nil {
		return ""
	}
	if e.NodeID != "" && e.Code != "" {
		return fmt.Sprintf("execution failure node=%s (%s): %s", e.NodeID, e.Code, e.Message)
	}
	if e.NodeID != "" {
		return fmt.Sprintf("execution failure node=%s: %s", e.NodeID, e.Message)
	}
	return fmt.Sprintf("execution failure: %s", e.Message)
}

func (e *ExecutionFailureError) Unwrap() error { return e.Cause }

// SystemFailureError represents crashes/termination/system-level failures.
// Resumable (per frozen spec, assuming checkpoints exist).
type SystemFailureError struct {
	Code    string
	Message string
	Cause   error
}

func (e *SystemFailureError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("system failure (%s): %s", e.Code, e.Message)
	}
	return fmt.Sprintf("system failure: %s", e.Message)
}

func (e *SystemFailureError) Unwrap() error { return e.Cause }

func failureFromError(err error) (Failure, error) {
	if err == nil {
		return Failure{}, errors.New("nil error")
	}

	var gf *GraphFailureError
	if errors.As(err, &gf) && gf != nil {
		return Failure{
			FailureClass: FailureClassGraph,
			NodeID:       nil,
			ErrorCode:    nonEmptyOr(gf.Code, "GraphFailure"),
			ErrorMessage: nonEmptyOr(gf.Message, gf.Error()),
			Resumable:    false,
		}, nil
	}

	var wf *WorkspaceFailureError
	if errors.As(err, &wf) && wf != nil {
		return Failure{
			FailureClass: FailureClassWorkspace,
			NodeID:       nil,
			ErrorCode:    nonEmptyOr(wf.Code, "WorkspaceFailure"),
			ErrorMessage: nonEmptyOr(wf.Message, wf.Error()),
			Resumable:    false,
		}, nil
	}

	var ef *ExecutionFailureError
	if errors.As(err, &ef) && ef != nil {
		var nodePtr *string
		if ef.NodeID != "" {
			n := ef.NodeID
			nodePtr = &n
		}
		return Failure{
			FailureClass: FailureClassExecution,
			NodeID:       nodePtr,
			ErrorCode:    nonEmptyOr(ef.Code, "ExecutionFailure"),
			ErrorMessage: nonEmptyOr(ef.Message, ef.Error()),
			// Conditionally resumable; the caller decides based on checkpoint presence.
			Resumable: true,
		}, nil
	}

	var sf *SystemFailureError
	if errors.As(err, &sf) && sf != nil {
		return Failure{
			FailureClass: FailureClassSystem,
			NodeID:       nil,
			ErrorCode:    nonEmptyOr(sf.Code, "SystemFailure"),
			ErrorMessage: nonEmptyOr(sf.Message, sf.Error()),
			Resumable:    true,
		}, nil
	}

	// Unknown error: classify as system failure (most conservative within the 4-class taxonomy).
	return Failure{
		FailureClass: FailureClassSystem,
		NodeID:       nil,
		ErrorCode:    "UnknownError",
		ErrorMessage: err.Error(),
		Resumable:    true,
	}, nil
}

func nonEmptyOr(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
