package state

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// FailureRecorder writes failure.json artifacts for runs.
//
// It is intentionally small: callers provide Run metadata and the triggering error.
// The recorder classifies the error into the frozen failure taxonomy and persists
// the Failure record using Store (atomic + durable).
type FailureRecorder struct {
	Store *Store
}

func (r *FailureRecorder) NewRunID() (string, error) {
	// Run IDs are operational identifiers. The frozen sprint-08 spec does not define
	// a deterministic format, so we use a random 128-bit hex string.
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (r *FailureRecorder) StartRun(run Run) error {
	if r == nil || r.Store == nil {
		return errors.New("Store is required")
	}
	if run.StartTime.IsZero() {
		run.StartTime = time.Now().UTC()
	}
	if err := run.Validate(); err != nil {
		return fmt.Errorf("invalid run: %w", err)
	}
	return r.Store.SaveRun(run)
}

func (r *FailureRecorder) RecordFailure(runID string, err error) error {
	if r == nil || r.Store == nil {
		return errors.New("Store is required")
	}
	f, ferr := failureFromError(err)
	if ferr != nil {
		return ferr
	}
	return r.Store.SaveFailure(runID, f)
}
