package tui

import (
	"strings"
	"testing"
)

func TestIncompleteEscape(t *testing.T) {
	for _, c := range []struct {
		buffer string
		want   bool
	}{
		// Complete sequences: nothing to wait for
		{"\x1b[A", false},
		{"\x1bOA", false},
		{"\x1b[1;5A", false},
		{"\x1b[200~", false},
		{"\x1b[<0;1;1M", false},
		{"\x1b[12;34R", false},
		{"\x1b[?2004;2$y", false},
		{"\x1b[?1;2c", false},

		// Fragments: keep waiting
		{"\x1b[", true},
		{"\x1b[?", true},
		{"\x1b[1;", true},
		{"\x1b[?2004;2$", true},
		{"\x1bO", true},
		{"\x1b[<0;1;", true},

		// Only the trailing sequence matters
		{"ab\x1b[?2004;2$", true},
		{"\x1b[A\x1b[", true},
		{"\x1b[A\x1b[B", false},

		// Long buffers: only the tail is scanned, so an introducer further
		// back than escapeLookback is not waited for
		{strings.Repeat("a", 100000), false},
		{"\x1b[" + strings.Repeat("a", 100000), false},
		{strings.Repeat("a", 100000) + "\x1b[1;", true},

		// Not a sequence fzf waits on
		{"", false},
		{"abc", false},
		{"\x1b", false},      // lone ESC, handled by the existing escDelay branch
		{"\x1ba", false},     // ALT-a
		{"\x1b[\x01", false}, // malformed, do not stall on it
	} {
		if got := incompleteEscape([]byte(c.buffer)); got != c.want {
			t.Errorf("incompleteEscape(%q) = %v, want %v", c.buffer, got, c.want)
		}
	}
}

// A string sequence is not waited for. Its introducer is also ALT-], ALT-P or
// ALT-_, and holding the read open on those both delayed the key and let typed
// bytes accumulate into something that looked like a sequence.
func TestIncompleteStringEscape(t *testing.T) {
	for _, buffer := range []string{
		"\x1b]11;rgb:",
		"\x1bP>|kitty",
		"\x1b_Gi=1",
		"\x1b]0;t\x1b",
		"\x1b]0;title\a",
		"ab\x1b]11;rgb:",
	} {
		if incompleteEscape([]byte(buffer)) {
			t.Errorf("incompleteEscape(%q) = true, want false", buffer)
		}
	}
}

// SS3 reaches the same parameterized subcases as CSI in the parser, so it has to
// be judged the same way here. Declaring it finished after one byte made the
// parser give up on a split \eO1;5A and type its tail into the query.
func TestIncompleteSS3(t *testing.T) {
	for _, c := range []struct {
		buffer string
		want   bool
	}{
		{"\x1bO", true},
		{"\x1bO1", true},
		{"\x1bO1;", true},
		{"\x1bO1;5", true},
		{"\x1bOA", false},
		{"\x1bOP", false},
		{"\x1bO1;5A", false},
		{"\x1b[23$", false}, // rxvt SHIFT-F11 is complete
		{"\x1b[7$", false},
		{"\x1b[4;2$", true}, // $ after a ; is intermediate
	} {
		if got := incompleteEscape([]byte(c.buffer)); got != c.want {
			t.Errorf("incompleteEscape(%q) = %v, want %v", c.buffer, got, c.want)
		}
	}
}
