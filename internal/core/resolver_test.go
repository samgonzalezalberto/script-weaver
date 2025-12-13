package core

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolve_StrictlySorted verifies tdd.md#Test-9:
// "Expanded file list MUST be strictly sorted.
// Different filesystem ordering MUST NOT affect hashing or execution."
func TestResolve_StrictlySorted(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "resolver-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files in non-alphabetical order to test sorting
	// Filesystem may return these in any order depending on OS
	files := []string{"zebra.txt", "apple.txt", "mango.txt", "banana.txt"}
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content-"+name), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", name, err)
		}
	}

	resolver := NewInputResolver(tmpDir)
	result, err := resolver.Resolve([]string{"*.txt"})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify strict sorting
	if len(result.Inputs) != 4 {
		t.Fatalf("expected 4 inputs, got %d", len(result.Inputs))
	}

	expectedOrder := []string{"apple.txt", "banana.txt", "mango.txt", "zebra.txt"}
	for i, expected := range expectedOrder {
		actual := filepath.Base(result.Inputs[i].Path)
		if actual != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, actual)
		}
	}
}

// TestResolve_ContentBasedIdentity verifies spec.md Input Determinism:
// "Input files are read by content."
func TestResolve_ContentBasedIdentity(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolver-content-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	expectedContent := []byte("file content for identity")
	filePath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(filePath, expectedContent, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	resolver := NewInputResolver(tmpDir)
	result, err := resolver.Resolve([]string{"input.txt"})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.Inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(result.Inputs))
	}

	if string(result.Inputs[0].Content) != string(expectedContent) {
		t.Errorf("content mismatch: expected %q, got %q",
			expectedContent, result.Inputs[0].Content)
	}
}

// TestResolve_DeterministicAcrossRuns verifies that multiple resolutions
// of the same patterns produce identical results.
func TestResolve_DeterministicAcrossRuns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolver-determinism-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple files
	for i := 0; i < 10; i++ {
		name := string(rune('a'+i)) + ".txt"
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(name), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	resolver := NewInputResolver(tmpDir)
	patterns := []string{"*.txt"}

	// Resolve multiple times
	var results []*InputSet
	for i := 0; i < 5; i++ {
		result, err := resolver.Resolve(patterns)
		if err != nil {
			t.Fatalf("Resolve iteration %d failed: %v", i, err)
		}
		results = append(results, result)
	}

	// All results must be identical
	first := results[0]
	for i := 1; i < len(results); i++ {
		if len(results[i].Inputs) != len(first.Inputs) {
			t.Errorf("iteration %d: different input count", i)
			continue
		}
		for j := range first.Inputs {
			if results[i].Inputs[j].Path != first.Inputs[j].Path {
				t.Errorf("iteration %d, input %d: path mismatch", i, j)
			}
			if string(results[i].Inputs[j].Content) != string(first.Inputs[j].Content) {
				t.Errorf("iteration %d, input %d: content mismatch", i, j)
			}
		}
	}
}

// TestResolve_DeduplicatesOverlappingPatterns verifies that overlapping
// patterns do not produce duplicate inputs.
func TestResolve_DeduplicatesOverlappingPatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolver-dedup-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	resolver := NewInputResolver(tmpDir)
	// Both patterns match the same file
	result, err := resolver.Resolve([]string{"*.txt", "file.txt"})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.Inputs) != 1 {
		t.Errorf("expected 1 input (deduplicated), got %d", len(result.Inputs))
	}
}

// TestResolve_EmptyPatterns returns empty InputSet.
func TestResolve_EmptyPatterns(t *testing.T) {
	resolver := NewInputResolver("/tmp")
	result, err := resolver.Resolve([]string{})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.Inputs) != 0 {
		t.Errorf("expected 0 inputs, got %d", len(result.Inputs))
	}
}

// TestResolve_SkipsDirectories verifies that directories are not included.
func TestResolve_SkipsDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolver-skipdir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory and a file
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	resolver := NewInputResolver(tmpDir)
	result, err := resolver.Resolve([]string{"*"})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Should only have the file, not the directory
	if len(result.Inputs) != 1 {
		t.Errorf("expected 1 input (file only), got %d", len(result.Inputs))
	}
}

// TestResolve_NormalizesPathSeparators verifies cross-platform path handling.
func TestResolve_NormalizesPathSeparators(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolver-paths-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested structure
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	filePath := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	resolver := NewInputResolver(tmpDir)
	result, err := resolver.Resolve([]string{"sub/*.txt"})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.Inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(result.Inputs))
	}

	// Path should use forward slashes for cross-platform determinism
	path := result.Inputs[0].Path
	for _, c := range path {
		if c == '\\' {
			t.Errorf("path contains backslash (not normalized): %s", path)
			break
		}
	}
}
