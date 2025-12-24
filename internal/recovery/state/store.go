package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store provides persistent storage for execution state under:
//   <baseDir>/.scriptweaver/runs/<run-id>/
//
// All state writes are atomic and durable (file sync + atomic rename + dir sync).
type Store struct {
	baseDir string
}

func NewStore(baseDir string) (*Store, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, errors.New("baseDir is required")
	}
	return &Store{baseDir: baseDir}, nil
}

func (s *Store) runsRootDir() string {
	return filepath.Join(s.baseDir, ".scriptweaver", "runs")
}

// ListRunIDs returns all run IDs currently present on disk.
//
// Determinism: the returned slice is sorted lexicographically.
func (s *Store) ListRunIDs() ([]string, error) {
	if s == nil {
		return nil, errors.New("nil Store")
	}
	root := s.runsRootDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.TrimSpace(e.Name())
		if name == "" {
			continue
		}
		ids = append(ids, name)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *Store) runDir(runID string) string {
	return filepath.Join(s.runsRootDir(), runID)
}

func (s *Store) runPath(runID string) string {
	return filepath.Join(s.runDir(runID), "run.json")
}

func (s *Store) failurePath(runID string) string {
	return filepath.Join(s.runDir(runID), "failure.json")
}

func (s *Store) checkpointsDir(runID string) string {
	return filepath.Join(s.runDir(runID), "checkpoints")
}

func (s *Store) checkpointPath(runID, nodeID string) string {
	// Assumption (documented in notes): node_id is a stable identifier safe to use as a filename.
	return filepath.Join(s.checkpointsDir(runID), nodeID+".json")
}

// LoadAllCheckpoints loads all checkpoint records for a given run.
//
// Determinism: returned map values are loaded from files discovered via sorted directory listing.
func (s *Store) LoadAllCheckpoints(runID string) (map[string]Checkpoint, error) {
	if s == nil {
		return nil, errors.New("nil Store")
	}
	if strings.TrimSpace(runID) == "" {
		return nil, errors.New("runID is required")
	}
	dir := s.checkpointsDir(runID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Checkpoint{}, nil
		}
		return nil, err
	}
	// os.ReadDir returns entries sorted by filename.
	out := make(map[string]Checkpoint, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		nodeID := strings.TrimSuffix(name, ".json")
		if strings.TrimSpace(nodeID) == "" {
			continue
		}
		cp, err := s.LoadCheckpoint(runID, nodeID)
		if err != nil {
			return nil, err
		}
		out[nodeID] = cp
	}
	return out, nil
}

func (s *Store) SaveRun(run Run) error {
	if err := run.Validate(); err != nil {
		return fmt.Errorf("invalid run: %w", err)
	}
	if err := ensureDirDurable(s.runDir(run.RunID), 0o755); err != nil {
		return fmt.Errorf("ensure run dir: %w", err)
	}
	data, err := jsonMarshalStable(run)
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	if err := writeFileAtomicDurable(s.runPath(run.RunID), data, 0o644); err != nil {
		return fmt.Errorf("write run: %w", err)
	}
	return nil
}

func (s *Store) LoadRun(runID string) (Run, error) {
	var run Run
	if strings.TrimSpace(runID) == "" {
		return Run{}, errors.New("runID is required")
	}
	if err := readJSONStrict(s.runPath(runID), &run); err != nil {
		return Run{}, err
	}
	if err := run.Validate(); err != nil {
		return Run{}, fmt.Errorf("invalid run on disk: %w", err)
	}
	return run, nil
}

func (s *Store) SaveCheckpoint(runID string, checkpoint Checkpoint) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("runID is required")
	}
	if err := checkpoint.Validate(); err != nil {
		return fmt.Errorf("invalid checkpoint: %w", err)
	}
	// Ensure cache_keys is serialized as [] rather than null.
	if checkpoint.CacheKeys == nil {
		checkpoint.CacheKeys = []string{}
	}

	if err := ensureDirDurable(s.checkpointsDir(runID), 0o755); err != nil {
		return fmt.Errorf("ensure checkpoints dir: %w", err)
	}
	data, err := jsonMarshalStable(checkpoint)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	if err := writeFileAtomicDurable(s.checkpointPath(runID, checkpoint.NodeID), data, 0o644); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	return nil
}

func (s *Store) LoadCheckpoint(runID, nodeID string) (Checkpoint, error) {
	var checkpoint Checkpoint
	if strings.TrimSpace(runID) == "" {
		return Checkpoint{}, errors.New("runID is required")
	}
	if strings.TrimSpace(nodeID) == "" {
		return Checkpoint{}, errors.New("nodeID is required")
	}
	if err := readJSONStrict(s.checkpointPath(runID, nodeID), &checkpoint); err != nil {
		return Checkpoint{}, err
	}
	if checkpoint.CacheKeys == nil {
		return Checkpoint{}, errors.New("invalid checkpoint on disk: cache_keys must be an array (not null)")
	}
	if err := checkpoint.Validate(); err != nil {
		return Checkpoint{}, fmt.Errorf("invalid checkpoint on disk: %w", err)
	}
	return checkpoint, nil
}

func (s *Store) SaveFailure(runID string, failure Failure) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("runID is required")
	}
	if err := failure.Validate(); err != nil {
		return fmt.Errorf("invalid failure: %w", err)
	}
	if err := ensureDirDurable(s.runDir(runID), 0o755); err != nil {
		return fmt.Errorf("ensure run dir: %w", err)
	}
	data, err := jsonMarshalStable(failure)
	if err != nil {
		return fmt.Errorf("marshal failure: %w", err)
	}
	if err := writeFileAtomicDurable(s.failurePath(runID), data, 0o644); err != nil {
		return fmt.Errorf("write failure: %w", err)
	}
	return nil
}

func (s *Store) LoadFailure(runID string) (Failure, error) {
	var failure Failure
	if strings.TrimSpace(runID) == "" {
		return Failure{}, errors.New("runID is required")
	}
	if err := readJSONStrict(s.failurePath(runID), &failure); err != nil {
		return Failure{}, err
	}
	if err := failure.Validate(); err != nil {
		return Failure{}, fmt.Errorf("invalid failure on disk: %w", err)
	}
	return failure, nil
}

func jsonMarshalStable(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func readJSONStrict(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Ensure no trailing junk.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid JSON: trailing content")
	}
	return nil
}

func ensureDirDurable(dir string, perm os.FileMode) error {
	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}
	// Best-effort durability: sync the directory and its parent.
	if err := fsyncDir(dir); err != nil {
		return err
	}
	parent := filepath.Dir(dir)
	if parent != dir {
		if err := fsyncDir(parent); err != nil {
			return err
		}
	}
	return nil
}

func writeFileAtomicDurable(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	// Write all bytes.
	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	committed = true
	return fsyncDir(dir)
}

func fsyncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
