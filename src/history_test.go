package fzf

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"
)

func TestHistory(t *testing.T) {
	maxHistory := 50

	// Invalid arguments
	var paths []string
	if runtime.GOOS == "windows" {
		// GOPATH should exist, so we shouldn't be able to override it
		paths = []string{os.Getenv("GOPATH")}
	} else {
		paths = []string{"/etc", "/proc"}
	}

	for _, path := range paths {
		if _, e := NewHistory(path, maxHistory); e == nil {
			t.Error("Error expected for: " + path)
		}
	}

	f, _ := ioutil.TempFile("", "fzf-history")
	f.Close()

	{ // Append lines
		h, _ := NewHistory(f.Name(), maxHistory)
		for i := 0; i < maxHistory+10; i++ {
			h.append("foobar")
		}
	}
	{ // Read lines
		h, _ := NewHistory(f.Name(), maxHistory)
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
		h, _ := NewHistory(f.Name(), maxHistory)
		h.append("barfoo")
		h.append("")
		h.append("foobarbaz")
	}
	{ // Read lines again
		h, _ := NewHistory(f.Name(), maxHistory)
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
