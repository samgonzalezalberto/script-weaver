package state

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"scriptweaver/internal/incremental"
	"scriptweaver/internal/projectintegration/engine/workspace"
)

// ResumeEligibilityChecker determines whether a new run may resume from a previous run.
//
// Enforces frozen sprint-08 Resume Eligibility Rules:
//   - Graph hash unchanged
//   - Workspace intact and validated
//   - previous_run_id linked and exists
//   - No upstream invalidation markers exist
//
// Also enforces Run Retry semantics:
//   - retry_count is incremented when retrying a failed run
//   - previous_run_id points to the failed run being retried
type ResumeEligibilityChecker struct {
	Store       *Store
	ProjectRoot string
}

type ResumeEligibilityRequest struct {
	// NewRun is the candidate run attempting to resume.
	NewRun Run

	// ResumeFromNodeID is the checkpoint node the engine intends to resume from.
	ResumeFromNodeID string

	// Graph and Invalidation are the current invalidation markers and graph structure
	// used to verify that no upstream invalidation exists.
	Graph        *incremental.GraphSnapshot
	Invalidation incremental.InvalidationMap
}

func (c *ResumeEligibilityChecker) Check(req ResumeEligibilityRequest) error {
	if c == nil {
		return errors.New("nil ResumeEligibilityChecker")
	}
	if c.Store == nil {
		return errors.New("Store is required")
	}
	if strings.TrimSpace(c.ProjectRoot) == "" {
		return errors.New("ProjectRoot is required")
	}
	if err := req.NewRun.Validate(); err != nil {
		return fmt.Errorf("invalid new run: %w", err)
	}

	// Resume must not be attempted in clean mode.
	switch req.NewRun.Mode {
	case ExecutionModeIncremental, ExecutionModeResumeOnly:
		// ok
	default:
		return fmt.Errorf("resume not permitted in mode %q", req.NewRun.Mode)
	}

	// Workspace must validate (no corruption / unauthorized entries).
	if _, err := workspace.EnsureWorkspace(c.ProjectRoot); err != nil {
		return fmt.Errorf("workspace validation failed: %w", err)
	}

	// previous_run_id must be supplied and exist.
	if req.NewRun.PreviousRunID == nil || strings.TrimSpace(*req.NewRun.PreviousRunID) == "" {
		return errors.New("previous_run_id is required for resume")
	}
	prevID := strings.TrimSpace(*req.NewRun.PreviousRunID)
	prevRun, err := c.Store.LoadRun(prevID)
	if err != nil {
		return fmt.Errorf("previous run does not exist: %w", err)
	}

	// Graph hash must be unchanged.
	if prevRun.GraphHash != req.NewRun.GraphHash {
		return fmt.Errorf("graph hash mismatch (prev=%s new=%s)", prevRun.GraphHash, req.NewRun.GraphHash)
	}

	// Retry semantics: previous_run_id must point to a failed run being retried.
	prevFailure, ferr := c.Store.LoadFailure(prevID)
	if ferr != nil {
		if os.IsNotExist(ferr) {
			return errors.New("previous_run_id must point to a failed run (failure record missing)")
		}
		return fmt.Errorf("loading previous run failure: %w", ferr)
	}
	if !prevFailure.Resumable {
		return fmt.Errorf("previous run failure is not resumable (class=%s code=%s)", prevFailure.FailureClass, prevFailure.ErrorCode)
	}
	if req.NewRun.RetryCount != prevRun.RetryCount+1 {
		return fmt.Errorf("retry_count must be incremented (prev=%d new=%d)", prevRun.RetryCount, req.NewRun.RetryCount)
	}
	if prevRun.RunID != prevID {
		// Defensive sanity check; LoadRun is keyed by ID but keep it explicit.
		return errors.New("previous_run_id mismatch")
	}

	// No upstream invalidation markers exist.
	if strings.TrimSpace(req.ResumeFromNodeID) == "" {
		return errors.New("ResumeFromNodeID is required")
	}
	invalidatedUpstream, err := upstreamInvalidations(req.Graph, req.Invalidation, req.ResumeFromNodeID)
	if err != nil {
		return err
	}
	if len(invalidatedUpstream) != 0 {
		return fmt.Errorf("resume blocked by upstream invalidation: %s", strings.Join(invalidatedUpstream, ","))
	}

	return nil
}

func upstreamInvalidations(g *incremental.GraphSnapshot, inv incremental.InvalidationMap, nodeID string) ([]string, error) {
	if g == nil || g.Nodes == nil {
		return nil, errors.New("graph snapshot is required")
	}
	if _, ok := g.Nodes[nodeID]; !ok {
		return nil, fmt.Errorf("resume node %q not found in graph snapshot", nodeID)
	}
	if inv == nil {
		return nil, errors.New("invalidation map is required")
	}

	visited := map[string]bool{}
	stack := []string{nodeID}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[n] {
			continue
		}
		visited[n] = true

		snap, ok := g.Nodes[n]
		if !ok {
			continue
		}
		for _, up := range snap.Upstream {
			if strings.TrimSpace(up) == "" {
				continue
			}
			stack = append(stack, up)
		}
	}

	invalidated := make([]string, 0)
	for n := range visited {
		e, ok := inv[n]
		if !ok {
			return nil, fmt.Errorf("missing invalidation entry for %q", n)
		}
		if e.Invalidated {
			invalidated = append(invalidated, n)
		}
	}
	sort.Strings(invalidated)
	return invalidated, nil
}
