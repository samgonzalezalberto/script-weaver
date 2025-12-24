package state

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"
	"time"

	"scriptweaver/internal/core"
	"scriptweaver/internal/trace"
)

// CheckpointValidator validates and persists checkpoints.
//
// It enforces the Checkpoint Rules in the frozen sprint-08 execution recovery spec:
//   - Node execution completed successfully
//   - Outputs were written deterministically
//   - Cache entries are present and verified
//   - Trace entry is complete
//
// Checkpoint metadata persistence is atomic and durable via Store.
type CheckpointValidator struct {
	Store     *Store
	Cache     core.Cache
	Harvester *core.Harvester
}

type CheckpointInput struct {
	RunID    string
	NodeID   string
	When     time.Time
	TaskHash core.TaskHash

	DeclaredOutputs []string
	ExitCode        int
	FromCache       bool
	TraceEvents     []trace.TraceEvent
}

// CreateAndSave validates the provided evidence and, if valid, writes a checkpoint.
//
// Returns a descriptive error if any validation fails.
func (v *CheckpointValidator) CreateAndSave(in CheckpointInput) (Checkpoint, error) {
	if v == nil {
		return Checkpoint{}, errors.New("nil CheckpointValidator")
	}
	if v.Store == nil {
		return Checkpoint{}, errors.New("Store is required")
	}
	if v.Cache == nil {
		return Checkpoint{}, errors.New("Cache is required")
	}
	if v.Harvester == nil {
		return Checkpoint{}, errors.New("Harvester is required")
	}

	var errs []error
	if strings.TrimSpace(in.RunID) == "" {
		errs = append(errs, errors.New("runID is required"))
	}
	if strings.TrimSpace(in.NodeID) == "" {
		errs = append(errs, errors.New("nodeID is required"))
	}
	if in.When.IsZero() {
		errs = append(errs, errors.New("timestamp is required"))
	}
	if strings.TrimSpace(in.TaskHash.String()) == "" {
		errs = append(errs, errors.New("task hash is required"))
	}

	// 1) Verify node execution success.
	if in.ExitCode != 0 {
		errs = append(errs, fmt.Errorf("node did not succeed (exit_code=%d)", in.ExitCode))
	}

	// 2) Verify deterministic output writes by re-harvesting declared outputs and hashing.
	// Harvester guarantees stable path normalization and sorting.
	outputHash := ""
	if len(errs) == 0 { // avoid extra IO when already invalid
		artifactSet, err := v.Harvester.Harvest(in.DeclaredOutputs)
		if err != nil {
			errs = append(errs, fmt.Errorf("harvesting outputs: %w", err))
		} else {
			outputHash = computeArtifactSetHash(artifactSet)
			if strings.TrimSpace(outputHash) == "" {
				errs = append(errs, errors.New("output hash is empty"))
			}
		}
	}

	// 3) Verify cache entry existence.
	if len(errs) == 0 {
		exists, err := v.Cache.Has(in.TaskHash)
		if err != nil {
			errs = append(errs, fmt.Errorf("checking cache entry: %w", err))
		} else if !exists {
			errs = append(errs, fmt.Errorf("cache entry missing for task hash %s", in.TaskHash))
		}
	}

	// 4) Verify trace entry completion.
	if len(errs) == 0 {
		if err := validateTraceForCheckpoint(in.TraceEvents, in.NodeID, in.FromCache); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return Checkpoint{}, errors.Join(errs...)
	}

	cp := Checkpoint{
		NodeID:     in.NodeID,
		Timestamp:  in.When.UTC(),
		CacheKeys:  []string{in.TaskHash.String()},
		OutputHash: outputHash,
		Valid:      true,
	}
	if err := v.Store.SaveCheckpoint(in.RunID, cp); err != nil {
		return Checkpoint{}, err
	}
	return cp, nil
}

func validateTraceForCheckpoint(events []trace.TraceEvent, nodeID string, fromCache bool) error {
	seenFailed := false
	seenExecuted := false
	seenArtifactsRestored := false

	for _, e := range events {
		if e.TaskID != nodeID {
			continue
		}
		switch e.Kind {
		case trace.EventTaskFailed:
			seenFailed = true
		case trace.EventTaskExecuted:
			seenExecuted = true
		case trace.EventTaskArtifactsRestored:
			seenArtifactsRestored = true
		}
	}

	if seenFailed {
		return errors.New("trace indicates task failure")
	}
	if fromCache {
		if !seenArtifactsRestored && !seenExecuted {
			return errors.New("trace entry incomplete: expected TaskArtifactsRestored or TaskExecuted")
		}
		return nil
	}
	if !seenExecuted {
		return errors.New("trace entry incomplete: expected TaskExecuted")
	}
	return nil
}

func computeArtifactSetHash(set *core.ArtifactSet) string {
	// Deterministic hash over the harvested artifacts.
	h := sha256.New()
	if set == nil {
		h.Write([]byte("nil"))
		return hex.EncodeToString(h.Sum(nil))
	}
	for _, a := range set.Artifacts {
		writeLenPrefixed(h, []byte(a.Path))
		writeLenPrefixed(h, a.Content)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeLenPrefixed(h hash.Hash, b []byte) {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(b)))
	_, _ = h.Write(n[:])
	_, _ = h.Write(b)
}
