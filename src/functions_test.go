package fzf

import (
	"os"
	"strings"
	"testing"
)

func TestWriteTemporaryFile(t *testing.T) {
	// Test normal case
	data := []string{"line1", "line2", "line3"}
	filename := WriteTemporaryFile(data, "\n")
	if filename == "" {
		t.Fatal("Expected non-empty filename")
	}
	defer os.Remove(filename)

	// Verify file content
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	expected := "line1\nline2\nline3\n"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}

	// Test with different separator
	filename2 := WriteTemporaryFile(data, "\x00")
	if filename2 == "" {
		t.Fatal("Expected non-empty filename")
	}
	defer os.Remove(filename2)

	content2, err := os.ReadFile(filename2)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	expected2 := "line1\x00line2\x00line3\x00"
	if string(content2) != expected2 {
		t.Errorf("Expected %q, got %q", expected2, string(content2))
	}

	// Test empty data
	filename3 := WriteTemporaryFile([]string{}, "\n")
	if filename3 == "" {
		t.Fatal("Expected non-empty filename even for empty data")
	}
	defer os.Remove(filename3)

	content3, err := os.ReadFile(filename3)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if string(content3) != "\n" {
		t.Errorf("Expected newline for empty data, got %q", string(content3))
	}
}

func TestRemoveFiles(t *testing.T) {
	// Create temporary files
	f1, err := os.CreateTemp("", "fzf-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	f1.Close()
	f2, err := os.CreateTemp("", "fzf-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	f2.Close()

	// Verify files exist
	if _, err := os.Stat(f1.Name()); os.IsNotExist(err) {
		t.Fatal("File 1 should exist")
	}
	if _, err := os.Stat(f2.Name()); os.IsNotExist(err) {
		t.Fatal("File 2 should exist")
	}

	// Remove files
	removeFiles([]string{f1.Name(), f2.Name()})

	// Verify files are removed
	if _, err := os.Stat(f1.Name()); !os.IsNotExist(err) {
		t.Error("File 1 should be removed")
	}
	if _, err := os.Stat(f2.Name()); !os.IsNotExist(err) {
		t.Error("File 2 should be removed")
	}

	// Test removing non-existent file (should not panic)
	removeFiles([]string{"/nonexistent/path/to/file"})
}

func TestStringBytes(t *testing.T) {
	tests := []string{
		"hello",
		"",
		"unicode: 你好世界",
		"with\x00null",
		"special\n\r\tchars",
	}

	for _, input := range tests {
		result := stringBytes(input)
		if string(result) != input {
			t.Errorf("stringBytes(%q) = %q, expected %q", input, string(result), input)
		}
		if len(result) != len(input) {
			t.Errorf("len(stringBytes(%q)) = %d, expected %d", input, len(result), len(input))
		}
	}
}

func TestByteString(t *testing.T) {
	tests := [][]byte{
		[]byte("hello"),
		[]byte{},
		[]byte("unicode: 你好世界"),
		[]byte("with\x00null"),
		[]byte("special\n\r\tchars"),
	}

	for _, input := range tests {
		result := byteString(input)
		if string(result) != string(input) {
			t.Errorf("byteString(%q) = %q, expected %q", input, result, string(input))
		}
		if len(result) != len(input) {
			t.Errorf("len(byteString(%q)) = %d, expected %d", input, len(result), len(input))
		}
	}
}

func TestStringBytesByteStringRoundTrip(t *testing.T) {
	tests := []string{
		"hello world",
		"",
		"unicode: 你好世界 🎉",
		"with\x00null\x00bytes",
		"special\n\r\tchars\x1b[31mansi",
		strings.Repeat("long string ", 1000),
	}

	for _, input := range tests {
		// string -> bytes -> string
		bytes := stringBytes(input)
		result := byteString(bytes)
		if result != input {
			t.Errorf("Round-trip failed for %q: got %q", input, result)
		}
	}
}
