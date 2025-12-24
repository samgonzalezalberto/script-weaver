package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_SaveAndLoadRun_IncludesNullablePreviousRunID(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(base)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	run := Run{
		RunID:         "run-123",
		GraphHash:     "gh-abc",
		StartTime:     time.Unix(1, 2).UTC(),
		Mode:          ExecutionModeIncremental,
		RetryCount:    0,
		Status:        "running",
		PreviousRunID: nil,
	}
	if err := store.SaveRun(run); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	// Ensure JSON has previous_run_id: null (field must exist and be nullable).
	data, err := os.ReadFile(filepath.Join(base, ".scriptweaver", "runs", "run-123", "run.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "\"previous_run_id\": null") {
		t.Fatalf("expected previous_run_id to be null; got: %s", string(data))
	}

	loaded, err := store.LoadRun("run-123")
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	if loaded.RunID != run.RunID || loaded.GraphHash != run.GraphHash {
		t.Fatalf("loaded run mismatch: %+v", loaded)
	}
	if loaded.PreviousRunID != nil {
		t.Fatalf("expected PreviousRunID nil; got %v", *loaded.PreviousRunID)
	}
}

func TestStore_SaveAndLoadCheckpoint_CacheKeysNotNull(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(base)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	cp := Checkpoint{
		NodeID:     "A",
		Timestamp:  time.Unix(10, 0).UTC(),
		CacheKeys:  []string{"cache-key-1"},
		OutputHash: "out-hash-1",
		Valid:      true,
	}
	if err := store.SaveCheckpoint("run-1", cp); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".scriptweaver", "runs", "run-1", "checkpoints", "A.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "\"cache_keys\": null") {
		t.Fatalf("expected cache_keys array, got null: %s", string(data))
	}

	loaded, err := store.LoadCheckpoint("run-1", "A")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if loaded.NodeID != "A" || loaded.OutputHash != "out-hash-1" || !loaded.Valid {
		t.Fatalf("loaded checkpoint mismatch: %+v", loaded)
	}
}

func TestStore_SaveAndLoadFailure_NodeIDOptional(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(base)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	f := Failure{
		FailureClass: FailureClassSystem,
		NodeID:       nil,
		ErrorCode:    "SIGTERM",
		ErrorMessage: "terminated",
		Resumable:    true,
	}
	if err := store.SaveFailure("run-9", f); err != nil {
		t.Fatalf("SaveFailure: %v", err)
	}
	loaded, err := store.LoadFailure("run-9")
	if err != nil {
		t.Fatalf("LoadFailure: %v", err)
	}
	if loaded.FailureClass != FailureClassSystem || loaded.NodeID != nil || !loaded.Resumable {
		t.Fatalf("loaded failure mismatch: %+v", loaded)
	}
}
