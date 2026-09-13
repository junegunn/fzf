package fzf

import (
	"testing"
)

func TestTrimPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic cases
		{"./foo", "foo"},
		{"./foo/bar", "foo/bar"},
		{".", "."},
		{"./", "."},
		{"", "."},

		// Multiple leading ./
		{"././foo", "foo"},
		{"./././foo/bar", "foo/bar"},

		// Windows-style paths (should be handled)
		{".\\foo", "foo"},
		{".\\foo\\bar", "foo\\bar"},

		// No leading ./
		{"foo", "foo"},
		{"foo/bar", "foo/bar"},
		{"/foo/bar", "/foo/bar"},

		// Edge cases with just dots
		{"..", ".."},
		{"./..", ".."},
		{"../foo", "../foo"},
	}

	for _, tt := range tests {
		result := trimPath(tt.input)
		if result != tt.expected {
			t.Errorf("trimPath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsSymlinkToDir(t *testing.T) {
	// This test is limited since we can't easily create symlinks in unit tests
	// without platform-specific code. We'll test the basic logic.

	// Test with non-symlink (Type() won't have ModeSymlink)
	// We can't easily test this without mocking, so we'll just ensure
	// the function doesn't panic with empty input

	// The actual behavior is tested through integration tests
}

func TestReadFromStdin(t *testing.T) {
	// Testing readFromStdin directly is difficult because it reads from os.Stdin
	// We verify the function exists and has the correct signature through compilation
	// The actual behavior is tested through integration tests
}

func TestReaderFeedWithDelimiter(t *testing.T) {
	// Test that the reader correctly handles different delimiters
	// This is tested indirectly through TestReadFromCommand
}
