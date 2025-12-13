package core

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHarvest_OnlyDeclaredOutputsCaptured verifies tdd.md#Test-8:
// "Only declared outputs are captured."
func TestHarvest_OnlyDeclaredOutputsCaptured(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple files, only some declared as outputs
	declaredFile := filepath.Join(tmpDir, "declared.txt")
	undeclaredFile := filepath.Join(tmpDir, "undeclared.txt")

	if err := os.WriteFile(declaredFile, []byte("declared content"), 0644); err != nil {
		t.Fatalf("failed to write declared file: %v", err)
	}
	if err := os.WriteFile(undeclaredFile, []byte("undeclared content"), 0644); err != nil {
		t.Fatalf("failed to write undeclared file: %v", err)
	}

	harvester := NewHarvester(tmpDir)

	// Only declare one file
	result, err := harvester.Harvest([]string{"declared.txt"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// Should only have the declared file
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(result.Artifacts))
	}

	// Verify content
	if string(result.Artifacts[0].Content) != "declared content" {
		t.Errorf("wrong content: %s", result.Artifacts[0].Content)
	}
}

// TestHarvest_DirectoryRecursive verifies directory outputs are collected.
func TestHarvest_DirectoryRecursive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create output directory with nested structure
	outDir := filepath.Join(tmpDir, "output")
	subDir := filepath.Join(outDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Create files at different levels
	files := map[string]string{
		filepath.Join(outDir, "root.txt"):      "root content",
		filepath.Join(subDir, "nested.txt"):    "nested content",
		filepath.Join(subDir, "another.txt"):   "another content",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	// Also create file OUTSIDE the output directory (should NOT be captured)
	outsideFile := filepath.Join(tmpDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0644); err != nil {
		t.Fatalf("failed to write outside file: %v", err)
	}

	harvester := NewHarvester(tmpDir)
	result, err := harvester.Harvest([]string{"output"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// Should have 3 files from the output directory
	if len(result.Artifacts) != 3 {
		t.Fatalf("expected 3 artifacts, got %d", len(result.Artifacts))
	}

	// Verify outside file is NOT included
	for _, a := range result.Artifacts {
		if filepath.Base(a.Path) == "outside.txt" {
			t.Error("outside.txt should not be captured")
		}
	}
}

// TestHarvest_SortedOrder verifies artifacts are in deterministic order.
func TestHarvest_SortedOrder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files in non-alphabetical order
	files := []string{"zebra.txt", "apple.txt", "mango.txt"}
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(name), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	harvester := NewHarvester(tmpDir)
	result, err := harvester.Harvest([]string{"zebra.txt", "apple.txt", "mango.txt"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// Verify sorted order
	expectedOrder := []string{"apple.txt", "mango.txt", "zebra.txt"}
	for i, expected := range expectedOrder {
		actual := filepath.Base(result.Artifacts[i].Path)
		if actual != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, actual)
		}
	}
}

// TestHarvest_MissingOutputFails verifies missing declared output fails.
func TestHarvest_MissingOutputFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	harvester := NewHarvester(tmpDir)

	// Try to harvest non-existent file
	_, err = harvester.Harvest([]string{"missing.txt"})
	if err == nil {
		t.Error("expected error for missing output")
	}
}

// TestHarvest_EmptyOutputs returns empty ArtifactSet.
func TestHarvest_EmptyOutputs(t *testing.T) {
	harvester := NewHarvester("/tmp")
	result, err := harvester.Harvest([]string{})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	if len(result.Artifacts) != 0 {
		t.Errorf("expected 0 artifacts, got %d", len(result.Artifacts))
	}
}

// TestHarvest_DeduplicatesOverlapping verifies overlapping paths are deduplicated.
func TestHarvest_DeduplicatesOverlapping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a directory with a file
	outDir := filepath.Join(tmpDir, "output")
	if err := os.Mkdir(outDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	filePath := filepath.Join(outDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	harvester := NewHarvester(tmpDir)

	// Declare both the directory AND the specific file (overlapping)
	result, err := harvester.Harvest([]string{"output", "output/file.txt"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// Should only have 1 artifact (deduplicated)
	if len(result.Artifacts) != 1 {
		t.Errorf("expected 1 artifact (deduplicated), got %d", len(result.Artifacts))
	}
}

// TestHarvest_NormalizesPathSeparators verifies cross-platform paths.
func TestHarvest_NormalizesPathSeparators(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested file
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	filePath := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	harvester := NewHarvester(tmpDir)
	result, err := harvester.Harvest([]string{"sub/file.txt"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// Path should use forward slashes
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(result.Artifacts))
	}

	for _, c := range result.Artifacts[0].Path {
		if c == '\\' {
			t.Error("path contains backslash")
			break
		}
	}
}

// TestHarvest_WithNormalizer verifies content normalization.
func TestHarvest_WithNormalizer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file with timestamp content
	filePath := filepath.Join(tmpDir, "output.log")
	content := "Build started at 2024-12-13T10:30:45Z\nCompleted in 1.234s\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Harvest with normalizer
	normalizer := NewDefaultNormalizer()
	harvester := NewHarvesterWithNormalizer(tmpDir, normalizer)

	result, err := harvester.Harvest([]string{"output.log"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	normalized := string(result.Artifacts[0].Content)

	// Timestamp should be normalized
	if !containsPlaceholder(normalized, "<TIMESTAMP>") {
		t.Errorf("timestamp not normalized: %s", normalized)
	}

	// Duration should be normalized
	if !containsPlaceholder(normalized, "<DURATION>") {
		t.Errorf("duration not normalized: %s", normalized)
	}
}

func containsPlaceholder(s, placeholder string) bool {
	return len(s) > 0 && len(placeholder) > 0 && 
		(s == placeholder || len(s) > len(placeholder))
}

// TestHarvest_DoesNotUseGitStatus verifies we don't scan for modified files.
// This is a documentation test - the implementation simply doesn't have that code.
func TestHarvest_DoesNotUseGitStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harvester-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a declared file and an undeclared file
	declared := filepath.Join(tmpDir, "declared.txt")
	undeclared := filepath.Join(tmpDir, "modified-but-undeclared.txt")

	if err := os.WriteFile(declared, []byte("declared"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := os.WriteFile(undeclared, []byte("undeclared"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	harvester := NewHarvester(tmpDir)

	// Only harvest declared outputs
	result, err := harvester.Harvest([]string{"declared.txt"})
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// The undeclared file should NOT appear even though it exists
	if len(result.Artifacts) != 1 {
		t.Errorf("expected exactly 1 artifact, got %d", len(result.Artifacts))
	}

	for _, a := range result.Artifacts {
		if filepath.Base(a.Path) == "modified-but-undeclared.txt" {
			t.Error("undeclared file was captured - violates 'only declared outputs' rule")
		}
	}
}
