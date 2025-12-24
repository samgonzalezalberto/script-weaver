package state

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ExecutionMode string

const (
	ExecutionModeClean       ExecutionMode = "clean"
	ExecutionModeIncremental ExecutionMode = "incremental"
	ExecutionModeResumeOnly  ExecutionMode = "resume-only"
)

type RunStatus string

// Run is the persistent execution attempt metadata.
//
// Schema constraints (frozen): must include run_id, graph_hash, start_time, mode,
// retry_count, status, and previous_run_id (nullable).
type Run struct {
	RunID         string        `json:"run_id"`
	GraphHash     string        `json:"graph_hash"`
	StartTime     time.Time     `json:"start_time"`
	Mode          ExecutionMode `json:"mode"`
	RetryCount    int           `json:"retry_count"`
	Status        RunStatus     `json:"status"`
	PreviousRunID *string       `json:"previous_run_id"`
}

func (r Run) Validate() error {
	var errs []error
	if strings.TrimSpace(r.RunID) == "" {
		errs = append(errs, errors.New("run_id is required"))
	}
	if strings.TrimSpace(r.GraphHash) == "" {
		errs = append(errs, errors.New("graph_hash is required"))
	}
	if r.StartTime.IsZero() {
		errs = append(errs, errors.New("start_time is required"))
	}
	switch r.Mode {
	case ExecutionModeClean, ExecutionModeIncremental, ExecutionModeResumeOnly:
		// ok
	default:
		errs = append(errs, fmt.Errorf("invalid mode %q", r.Mode))
	}
	if r.RetryCount < 0 {
		errs = append(errs, errors.New("retry_count must be >= 0"))
	}
	if strings.TrimSpace(string(r.Status)) == "" {
		errs = append(errs, errors.New("status is required"))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// Checkpoint is a durable, validated execution boundary.
//
// Schema constraints (frozen): must include node_id, timestamp, cache_keys,
// output_hash, and valid.
type Checkpoint struct {
	NodeID     string    `json:"node_id"`
	Timestamp  time.Time `json:"timestamp"`
	CacheKeys  []string  `json:"cache_keys"`
	OutputHash string    `json:"output_hash"`
	Valid      bool      `json:"valid"`
}

func (c Checkpoint) Validate() error {
	var errs []error
	if strings.TrimSpace(c.NodeID) == "" {
		errs = append(errs, errors.New("node_id is required"))
	}
	if c.Timestamp.IsZero() {
		errs = append(errs, errors.New("timestamp is required"))
	}
	if c.CacheKeys == nil {
		errs = append(errs, errors.New("cache_keys must be an array (not null)"))
	}
	for i, k := range c.CacheKeys {
		if strings.TrimSpace(k) == "" {
			errs = append(errs, fmt.Errorf("cache_keys[%d] must not be empty", i))
		}
	}
	if strings.TrimSpace(c.OutputHash) == "" {
		errs = append(errs, errors.New("output_hash is required"))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

type FailureClass string

const (
	FailureClassGraph     FailureClass = "graph"
	FailureClassWorkspace FailureClass = "workspace"
	FailureClassExecution FailureClass = "execution"
	FailureClassSystem    FailureClass = "system"
)

// Failure is a recorded run termination reason.
//
// Schema constraints (frozen): must include failure_class, node_id (optional),
// error_code, error_message, and resumable.
type Failure struct {
	FailureClass FailureClass `json:"failure_class"`
	NodeID       *string      `json:"node_id,omitempty"`
	ErrorCode    string       `json:"error_code"`
	ErrorMessage string       `json:"error_message"`
	Resumable    bool         `json:"resumable"`
}

func (f Failure) Validate() error {
	var errs []error
	switch f.FailureClass {
	case FailureClassGraph, FailureClassWorkspace, FailureClassExecution, FailureClassSystem:
		// ok
	default:
		errs = append(errs, fmt.Errorf("invalid failure_class %q", f.FailureClass))
	}
	if f.NodeID != nil && strings.TrimSpace(*f.NodeID) == "" {
		errs = append(errs, errors.New("node_id must not be empty when provided"))
	}
	if strings.TrimSpace(f.ErrorCode) == "" {
		errs = append(errs, errors.New("error_code is required"))
	}
	if strings.TrimSpace(f.ErrorMessage) == "" {
		errs = append(errs, errors.New("error_message is required"))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
