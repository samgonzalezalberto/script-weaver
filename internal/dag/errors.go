package dag

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidGraph = errors.New("invalid task graph")
	ErrCycleFound   = errors.New("cycle detected")
)

// GraphError wraps deterministic graph validation failures.
type GraphError struct {
	Kind error
	Msg  string
}

func (e *GraphError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg == "" {
		return e.Kind.Error()
	}
	return fmt.Sprintf("%s: %s", e.Kind.Error(), e.Msg)
}

func (e *GraphError) Unwrap() error { return e.Kind }

func invalidf(format string, args ...any) error {
	return &GraphError{Kind: ErrInvalidGraph, Msg: fmt.Sprintf(format, args...)}
}

func cycleError(path []string) error {
	msg := "cycle"
	if len(path) > 0 {
		msg = "cycle: " + strings.Join(path, " -> ")
	}
	return &GraphError{Kind: ErrCycleFound, Msg: msg}
}
