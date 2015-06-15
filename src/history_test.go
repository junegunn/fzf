package fzf

import (
	"os/user"
	"testing"
)

func TestHistory(t *testing.T) {
	maxHistory := 50

	// Invalid arguments
	user, _ := user.Current()
	paths := []string{"/etc", "/proc"}
	if user.Name != "root" {
		paths = append(paths, "/etc/sudoers")
	}
	for _, path := range paths {
		if _, e := NewHistory(path, maxHistory); e == nil {
			t.Error("Error expected for: " + path)
		}
	}
	{ // Append lines
		h, _ := NewHistory("/tmp/fzf-history", maxHistory)
		for i := 0; i < maxHistory+10; i++ {
			h.append("foobar")
		}
	}
	{ // Read lines
		h, _ := NewHistory("/tmp/fzf-history", maxHistory)
		if len(h.lines) != maxHistory+1 {
			t.Errorf("Expected: %d, actual: %d\n", maxHistory+1, len(h.lines))
		}
		for i := 0; i < maxHistory; i++ {
			if h.lines[i] != "foobar" {
				t.Error("Expected: foobar, actual: " + h.lines[i])
			}
		}
	}
	{ // Append lines
		h, _ := NewHistory("/tmp/fzf-history", maxHistory)
		h.append("barfoo")
		h.append("")
		h.append("foobarbaz")
	}
	{ // Read lines again
		h, _ := NewHistory("/tmp/fzf-history", maxHistory)
		if len(h.lines) != maxHistory+1 {
			t.Errorf("Expected: %d, actual: %d\n", maxHistory+1, len(h.lines))
		}
		compare := func(idx int, exp string) {
			if h.lines[idx] != exp {
				t.Errorf("Expected: %s, actual: %s\n", exp, h.lines[idx])
			}
		}
		compare(maxHistory-3, "foobar")
		compare(maxHistory-2, "barfoo")
		compare(maxHistory-1, "foobarbaz")
	}
}
