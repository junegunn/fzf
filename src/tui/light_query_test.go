//go:build !windows

package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
)

// Drives queryStartup against a terminal simulated by pipes, with the replies
// already queued so the exchange is deterministic.
func replyingTerminal(t *testing.T, replies string) (*LightRenderer, func() string) {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inW.WriteString(replies); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inR.Close(); inW.Close(); outR.Close() })

	r := &LightRenderer{ttyin: inR, ttyout: outW, escDelay: defaultEscDelay}
	written := func() string {
		outW.Close()
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := outR.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		return sb.String()
	}
	return r, written
}

const eraseEcho = "\b \b"

func TestQueryStartup(t *testing.T) {
	for _, tc := range []struct {
		name     string
		replies  string
		row, col int
		paste    string // "", "true" or "false"
		erase    int    // columns the echo took, and so erasures expected
	}{
		{
			name:    "mode already set",
			replies: "\x1b[5;10R\x1b[?2004;1$y\x1b[5;10R",
			row:     4, col: 9, paste: "true",
		},
		{
			name:    "mode reset",
			replies: "\x1b[5;10R\x1b[?2004;2$y\x1b[5;10R",
			row:     4, col: 9, paste: "false",
		},
		{
			name:    "mode not recognized",
			replies: "\x1b[5;10R\x1b[?2004;0$y\x1b[5;10R",
			row:     4, col: 9,
		},
		{
			// Answers the position queries but ignores DECRQM without printing
			name:    "query ignored",
			replies: "\x1b[5;10R\x1b[5;10R",
			row:     4, col: 9,
		},
		{
			// macOS Terminal.app: ends the sequence at '$' and prints the 'p'
			name:    "query echoed",
			replies: "\x1b[5;10R\x1b[5;11R",
			row:     4, col: 9, erase: 1,
		},
		{
			// A terminal that prints more of what it could not parse
			name:    "longer echo",
			replies: "\x1b[5;10R\x1b[5;13R",
			row:     4, col: 9, erase: 3,
		},
		{
			// The echo wrapped to the next line, where erasing would guess
			name:    "echo wrapped",
			replies: "\x1b[5;80R\x1b[6;1R",
			row:     4, col: 79,
		},
		{
			name:    "no reply at all",
			replies: "",
			row:     -1, col: -1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, written := replyingTerminal(t, tc.replies)
			row, col, pasteWasSet := r.queryStartup()

			if row != tc.row || col != tc.col {
				t.Errorf("offset (%d,%d), want (%d,%d)", row, col, tc.row, tc.col)
			}
			paste := ""
			if pasteWasSet != nil {
				paste = "false"
				if *pasteWasSet {
					paste = "true"
				}
			}
			if paste != tc.paste {
				t.Errorf("pasteWasSet %q, want %q", paste, tc.paste)
			}

			out := written()
			if got := strings.Count(out, eraseEcho); got != tc.erase {
				t.Errorf("erased %d columns, want %d (wrote %q)", got, tc.erase, out)
			}
			// The paste mode query has to sit between two position queries
			if want := "\x1b[6n\x1b[?2004$p\x1b[6n"; !strings.Contains(out, want) {
				t.Errorf("queries %q, want %q in order", out, want)
			}
		})
	}
}

// The second-chance read has no time limit, so a key typed long after an
// unfinished sequence must not be dropped with it.
func TestLateKeyAfterUnfinishedSequence(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inR.Close(); inW.Close() })
	inW.WriteString("\x1b[12")
	go func() {
		time.Sleep(2 * defaultEscDelay * time.Millisecond)
		inW.WriteString("x")
		time.Sleep(2 * defaultEscDelay * time.Millisecond)
		inW.WriteString("y") // ends the wait if x was dropped
	}()

	r := &LightRenderer{ttyin: inR, escDelay: defaultEscDelay}
	for range 3 {
		if ev := r.GetChar(false); ev.Type == Rune {
			if ev.Char != 'x' {
				t.Errorf("got %q, want 'x'", ev.Char)
			}
			return
		}
	}
	t.Error("the late key was dropped")
}

// A read error during the second-chance read must end fzf cleanly, not panic.
func TestSecondChanceReadError(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inR.Close() })
	inW.WriteString("\x1b[12")
	go func() {
		time.Sleep(2 * defaultEscDelay * time.Millisecond)
		inW.Close()
	}()

	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	r := &LightRenderer{ttyin: inR, ttyout: out, escDelay: defaultEscDelay, origState: &term.State{}}
	if ev := r.GetChar(false); ev.Type != Fatal {
		t.Errorf("got %s, want Fatal", ev.Type.String())
	}
}
