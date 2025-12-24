package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scriptweaver/internal/core"
	"scriptweaver/internal/trace"
)

func TestCheckpointValidator_CreateAndSave_Success_Executed(t *testing.T) {
	base := t.TempDir()
	store, err := NewStore(base)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	cache := core.NewMemoryCache()
	h := core.NewHarvester(base)

	// Create deterministic output
	outPath := filepath.Join(base, "out.txt")
	if err := os.WriteFile(outPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Populate cache entry so Cache.Has returns true.
	hash := core.TaskHash("deadbeef")
	if err := cache.Put(&core.CacheEntry{Hash: hash, ExitCode: 0, Artifacts: []core.CachedArtifact{{Path: "out.txt", Content: []byte("hello")}}}); err != nil {
		t.Fatalf("cache.Put: %v", err)
	}

	v := &CheckpointValidator{Store: store, Cache: cache, Harvester: h}
	when := time.Unix(100, 0).UTC()

	cp, err := v.CreateAndSave(CheckpointInput{
		RunID:           "run-1",
		NodeID:          "A",
		When:            when,
		TaskHash:        hash,
		DeclaredOutputs: []string{"out.txt"},
		ExitCode:        0,
		FromCache:       false,
		TraceEvents:     []trace.TraceEvent{{Kind: trace.EventTaskExecuted, TaskID: "A", Reason: "FreshWork"}},
	})
	if err != nil {
		t.Fatalf("CreateAndSave: %v", err)
	}
	if !cp.Valid || cp.OutputHash == "" || cp.NodeID != "A" {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}

	loaded, err := store.LoadCheckpoint("run-1", "A")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if loaded.OutputHash != cp.OutputHash || !loaded.Valid {
		t.Fatalf("loaded checkpoint mismatch: %+v", loaded)
	}
}

func TestCheckpointValidator_CreateAndSave_Fails_WhenOutputsMissing(t *testing.T) {
	base := t.TempDir()
	store, _ := NewStore(base)
	cache := core.NewMemoryCache()
	h := core.NewHarvester(base)

	hash := core.TaskHash("deadbeef")
	_ = cache.Put(&core.CacheEntry{Hash: hash, ExitCode: 0})

	v := &CheckpointValidator{Store: store, Cache: cache, Harvester: h}
	_, err := v.CreateAndSave(CheckpointInput{
		RunID:           "run-1",
		NodeID:          "A",
		When:            time.Unix(1, 0).UTC(),
		TaskHash:        hash,
		DeclaredOutputs: []string{"missing.txt"},
		ExitCode:        0,
		FromCache:       false,
		TraceEvents:     []trace.TraceEvent{{Kind: trace.EventTaskExecuted, TaskID: "A"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	// Should not write checkpoint on invalid.
	if _, loadErr := store.LoadCheckpoint("run-1", "A"); loadErr == nil {
		t.Fatalf("expected no checkpoint persisted")
	}
}

func TestCheckpointValidator_CreateAndSave_Fails_WhenCacheMissing(t *testing.T) {
	base := t.TempDir()
	store, _ := NewStore(base)
	cache := core.NewMemoryCache()
	h := core.NewHarvester(base)

	if err := os.WriteFile(filepath.Join(base, "out.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v := &CheckpointValidator{Store: store, Cache: cache, Harvester: h}
	_, err := v.CreateAndSave(CheckpointInput{
		RunID:           "run-1",
		NodeID:          "A",
		When:            time.Unix(1, 0).UTC(),
		TaskHash:        core.TaskHash("missing"),
		DeclaredOutputs: []string{"out.txt"},
		ExitCode:        0,
		FromCache:       false,
		TraceEvents:     []trace.TraceEvent{{Kind: trace.EventTaskExecuted, TaskID: "A"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCheckpointValidator_CreateAndSave_Fails_WhenTraceIncomplete(t *testing.T) {
	base := t.TempDir()
	store, _ := NewStore(base)
	cache := core.NewMemoryCache()
	h := core.NewHarvester(base)

	if err := os.WriteFile(filepath.Join(base, "out.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	hash := core.TaskHash("deadbeef")
	_ = cache.Put(&core.CacheEntry{Hash: hash, ExitCode: 0})

	v := &CheckpointValidator{Store: store, Cache: cache, Harvester: h}
	_, err := v.CreateAndSave(CheckpointInput{
		RunID:           "run-1",
		NodeID:          "A",
		When:            time.Unix(1, 0).UTC(),
		TaskHash:        hash,
		DeclaredOutputs: []string{"out.txt"},
		ExitCode:        0,
		FromCache:       false,
		TraceEvents:     []trace.TraceEvent{{Kind: trace.EventTaskCached, TaskID: "A"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

// Ensure the validator can be used with a real Runner/Harvester setup (smoke).
func TestCheckpointValidator_Smoke_WithRunnerCache(t *testing.T) {
	base := t.TempDir()
	store, _ := NewStore(base)
	cache := core.NewMemoryCache()
	runner := core.NewRunner(base, cache)

	// Task that writes a deterministic output file.
	task := &core.Task{Name: "A", Run: "sh -c 'echo -n hi > out.txt'", Outputs: []string{"out.txt"}}
	res, err := runner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected success")
	}

	v := &CheckpointValidator{Store: store, Cache: cache, Harvester: runner.Harvester}
	_, err = v.CreateAndSave(CheckpointInput{
		RunID:           "run-1",
		NodeID:          "A",
		When:            time.Unix(2, 0).UTC(),
		TaskHash:        res.Hash,
		DeclaredOutputs: []string{"out.txt"},
		ExitCode:        0,
		FromCache:       false,
		TraceEvents:     []trace.TraceEvent{{Kind: trace.EventTaskExecuted, TaskID: "A"}},
	})
	if err != nil {
		t.Fatalf("CreateAndSave: %v", err)
	}
}
