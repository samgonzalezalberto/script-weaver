package core

import (
	"strings"
	"testing"
)

// TestDefaultNormalizer_ISO8601Timestamps verifies ISO 8601 timestamps are normalized.
func TestDefaultNormalizer_ISO8601Timestamps(t *testing.T) {
	n := NewDefaultNormalizer()

	testCases := []struct {
		input    string
		contains string
	}{
		{"Started at 2024-12-13T10:30:45Z", "<TIMESTAMP>"},
		{"Time: 2024-12-13T10:30:45.123Z", "<TIMESTAMP>"},
		{"2024-12-13T10:30:45+05:30 log entry", "<TIMESTAMP>"},
		{"2024-12-13T10:30:45-08:00", "<TIMESTAMP>"},
	}

	for _, tc := range testCases {
		result := string(n.Normalize([]byte(tc.input)))
		if !strings.Contains(result, tc.contains) {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.contains, result)
		}
		// Original timestamp should be gone
		if strings.Contains(result, "2024-12-13T") {
			t.Errorf("timestamp not removed from: %s", result)
		}
	}
}

// TestDefaultNormalizer_LogTimestamps verifies common log timestamps are normalized.
func TestDefaultNormalizer_LogTimestamps(t *testing.T) {
	n := NewDefaultNormalizer()

	testCases := []string{
		"2024-12-13 10:30:45 [INFO] message",
		"2024/12/13 10:30:45.123 log entry",
	}

	for _, input := range testCases {
		result := string(n.Normalize([]byte(input)))
		if !strings.Contains(result, "<TIMESTAMP>") {
			t.Errorf("log timestamp not normalized: %s -> %s", input, result)
		}
	}
}

// TestDefaultNormalizer_UnixTimestamps verifies Unix timestamps are normalized.
func TestDefaultNormalizer_UnixTimestamps(t *testing.T) {
	n := NewDefaultNormalizer()

	testCases := []string{
		"Created at 1702469445",
		"Timestamp: 1702469445123",
	}

	for _, input := range testCases {
		result := string(n.Normalize([]byte(input)))
		if !strings.Contains(result, "<UNIX_TS>") {
			t.Errorf("unix timestamp not normalized: %s -> %s", input, result)
		}
	}
}

// TestDefaultNormalizer_Durations verifies duration patterns are normalized.
func TestDefaultNormalizer_Durations(t *testing.T) {
	n := NewDefaultNormalizer()

	testCases := []string{
		"Completed in 1.234s",
		"Took 500ms",
		"Duration: 5 seconds",
		"Elapsed: 2 minutes",
	}

	for _, input := range testCases {
		result := string(n.Normalize([]byte(input)))
		if !strings.Contains(result, "<DURATION>") {
			t.Errorf("duration not normalized: %s -> %s", input, result)
		}
	}
}

// TestDefaultNormalizer_ProcessIDs verifies PIDs are normalized.
func TestDefaultNormalizer_ProcessIDs(t *testing.T) {
	n := NewDefaultNormalizer()

	testCases := []string{
		"pid 12345",
		"PID: 98765",
		"Process PID 54321 started",
	}

	for _, input := range testCases {
		result := string(n.Normalize([]byte(input)))
		if !strings.Contains(result, "<PID>") {
			t.Errorf("PID not normalized: %s -> %s", input, result)
		}
	}
}

// TestDefaultNormalizer_MemoryAddresses verifies addresses are normalized.
func TestDefaultNormalizer_MemoryAddresses(t *testing.T) {
	n := NewDefaultNormalizer()

	testCases := []string{
		"Pointer: 0x7fff5fbff8c0",
		"Address 0x0000000000401000",
	}

	for _, input := range testCases {
		result := string(n.Normalize([]byte(input)))
		if !strings.Contains(result, "<ADDR>") {
			t.Errorf("address not normalized: %s -> %s", input, result)
		}
	}
}

// TestDefaultNormalizer_PreservesNonMatching verifies non-matching content is preserved.
func TestDefaultNormalizer_PreservesNonMatching(t *testing.T) {
	n := NewDefaultNormalizer()

	input := "Hello, World! This is a normal message."
	result := string(n.Normalize([]byte(input)))

	if result != input {
		t.Errorf("non-matching content changed: %q -> %q", input, result)
	}
}

// TestDefaultNormalizer_MultiplePatterns verifies multiple patterns in one string.
func TestDefaultNormalizer_MultiplePatterns(t *testing.T) {
	n := NewDefaultNormalizer()

	input := "2024-12-13T10:30:45Z pid 12345 took 1.5s at 0x7fff5fbff8c0"
	result := string(n.Normalize([]byte(input)))

	expected := []string{"<TIMESTAMP>", "<PID>", "<DURATION>", "<ADDR>"}
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("expected %q in result: %s", exp, result)
		}
	}
}

// TestDefaultNormalizer_Deterministic verifies same input produces same output.
func TestDefaultNormalizer_Deterministic(t *testing.T) {
	n := NewDefaultNormalizer()

	input := []byte("2024-12-13T10:30:45Z pid 12345 took 1.5s")

	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		results[i] = string(n.Normalize(input))
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("non-deterministic: %q != %q", results[i], results[0])
		}
	}
}

// TestRawNormalizer_PreservesContent verifies raw normalizer is pass-through.
func TestRawNormalizer_PreservesContent(t *testing.T) {
	n := NewRawNormalizer()

	input := []byte("2024-12-13T10:30:45Z unchanged content\r\n")
	result := n.Normalize(input)

	if string(result) != string(input) {
		t.Errorf("raw normalizer changed content: %q -> %q", input, result)
	}
}

// TestStreamNormalizer_ConvertsLineEndings verifies CRLF to LF conversion.
func TestStreamNormalizer_ConvertsLineEndings(t *testing.T) {
	n := NewStreamNormalizer(nil)

	input := []byte("line1\r\nline2\r\nline3\r\n")
	expected := "line1\nline2\nline3\n"

	result := string(n.Normalize(input))

	if result != expected {
		t.Errorf("line endings not converted: %q -> %q", input, result)
	}
}

// TestStreamNormalizer_WithInner verifies chained normalization.
func TestStreamNormalizer_WithInner(t *testing.T) {
	inner := NewDefaultNormalizer()
	n := NewStreamNormalizer(inner)

	input := []byte("2024-12-13T10:30:45Z\r\n")
	result := string(n.Normalize(input))

	// Should have LF, not CRLF
	if strings.Contains(result, "\r\n") {
		t.Error("CRLF not converted")
	}

	// Should have normalized timestamp
	if !strings.Contains(result, "<TIMESTAMP>") {
		t.Error("timestamp not normalized")
	}
}

// TestNormalization_IdenticalAcrossRuns verifies tdd.md#Test-6:
// "Normalized cached output MUST be identical across runs."
func TestNormalization_IdenticalAcrossRuns(t *testing.T) {
	n := NewDefaultNormalizer()

	// Simulate two different runs with different timestamps
	run1 := "Build completed at 2024-12-13T10:30:45Z in 1.234s"
	run2 := "Build completed at 2024-12-14T15:45:30Z in 2.567s"

	normalized1 := string(n.Normalize([]byte(run1)))
	normalized2 := string(n.Normalize([]byte(run2)))

	// After normalization, both should be identical
	if normalized1 != normalized2 {
		t.Errorf("normalized outputs differ:\nrun1: %s\nrun2: %s", normalized1, normalized2)
	}
}
