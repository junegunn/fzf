package tui

import "testing"

// An unrecognized escape sequence must be consumed whole, CSI and string
// sequences alike. Consuming only part of one leaves the rest to be read as
// input and typed into the query.
func TestUnknownEscapeSequence(t *testing.T) {
	for _, c := range []struct {
		sequence string
		event    EventType
		size     int
	}{
		// Key encodings fzf does not implement
		{"\x1b[97;5u", Invalid, 7},
		{"\x1b[70;5u", Invalid, 7}, // not Home, which its prefix \e[7 matches
		{"\x1b[42;5u", Invalid, 7}, // nor End
		{"\x1b[4;5~", Invalid, 6},
		{"\x1b[127;5u", Invalid, 8},
		{"\x1b[27;5;127~", Invalid, 11},
		{"\x1b[57441;1u", Invalid, 10},
		{"\x1b\x1b[97;5u", Invalid, 7}, // ALT prefixed, the first ESC is dropped

		// Replies to queries fzf did not send, or sent and stopped waiting for
		{"\x1b[?1;2c", Invalid, 7},
		{"\x1b[>0;95;0c", Invalid, 10},

		// Mouse report arriving while mouse input is off
		{"\x1b[<0;1;1M", Invalid, 9},

		// String sequences
		{"\x1b]11;rgb:4a4a/4a4a/4a4a\x1b\\", Invalid, 25}, // background color reply
		{"\x1b]0;a title\a", Invalid, 12},                 // BEL terminated
		{"\x1bP>|kitty(0.48.2)\x1b\\", Invalid, 19},       // XTVERSION reply
		{"\x1b_Gi=1;OK\x1b\\", Invalid, 11},               // Kitty graphics reply
		{"\x1bP1$r\a\x1b\\", Invalid, 8},                  // BEL is payload in a DCS
		{"\x1b]11;x\x1bX\x1b\\", Invalid, 6},              // ESC ends it, framing "\e]11;x"

		// SS3 is framed like CSI, since the parser routes both the same way
		{"\x1bO9;5u", Invalid, 6},
		{"\x1bO1;5X", Invalid, 6}, // unknown final byte
		{"\x1bO1;5A", CtrlUp, 6},  // recognized, unchanged
		{"\x1bOP", F1, 3},

		// Left alone: this is how ALT-[, ALT-O, ALT-], ALT-P and ALT-_ arrive
		{"\x1b[a", Alt, 2},
		{"\x1b[9A", Alt, 2}, // ALT-[ and two characters can look like a CSI
		{"\x1b[1m", Alt, 2},
		{"\x1b[ x", Alt, 2},
		{"\x1bOx", Alt, 2},
		{"\x1bO9A", Alt, 2},

		// rxvt keys fzf does not know: dropped, not typed
		{"\x1b[3^", Invalid, 4}, // CTRL-DELETE
		{"\x1b[5^", Invalid, 4}, // CTRL-PAGEUP
		{"\x1b[3@", Invalid, 4}, // CTRL-SHIFT-DELETE
		{"\x1b[11^", Invalid, 5},

		// rxvt ends keys with $ after digits, so what follows is typed
		{"\x1b[7$x", Home, 4}, // SHIFT-HOME, then x
		{"\x1b[7$1x", Home, 4},
		{"\x1b[3$x", Invalid, 4},
		{"\x1b[3$1x", Invalid, 4},
		{"\x1b[23$x", Invalid, 5}, // SHIFT-F11, then x
		{"\x1b[24$1x", Invalid, 5},

		// A parse that gives up past the frame consumes only the frame, so the
		// typed ~ after these stays in the buffer
		{"\x1b[3;1u~", Invalid, 6},
		{"\x1b[5;1u~", Invalid, 6},
		{"\x1b[2$~", Invalid, 4}, // rxvt SHIFT-INSERT
		{"\x1b[2^~", Invalid, 4}, // rxvt CTRL-INSERT
		{"\x1b[2A~", Invalid, 4},

		// Elsewhere $ and other intermediate bytes do not end a sequence
		{"\x1b[4;0;10;20;0&w", Invalid, 15}, // not End, which \e[4 matches
		{"\x1b[4;2$y", Invalid, 7},
		{"\x1b[2 q", Invalid, 5},
		{"\x1b]abc", Alt, 2},
		{"\x1b]11;rgb:", Alt, 2}, // terminator has not arrived
		{"\x1b]\x1b]", Alt, 2},   // a second sequence must not swallow the ALT key
		{"\x1b]\x1b[A", Alt, 2},
		{"\x1bP\x1bP", Alt, 2},
		{"\x1b_\x1b_", Alt, 2},
		{"\x1b]\x1b\\", Alt, 2}, // ALT-] then ALT-backslash, not an empty OSC
		{"\x1b]\a", Alt, 2},     // ALT-] then CTRL-G, which must still abort

		// Typed text after the introducer is not a reply, even when a
		// terminator follows in the same read
		{"\x1b]a\a", Alt, 2},
		{"\x1b]1\a", Alt, 2},
		{"\x1b]a\x1b[A", Alt, 2},
		{"\x1bPx\x1b[A", Alt, 2},
		{"\x1b_ab\x1bb", Alt, 2},

		// SOS and PM are not framed
		{"\x1bXsos\x1b\\", Alt, 2},
		{"\x1b^status\x1b\\", Alt, 2},

		// Left alone: no final byte yet, so the sequence may still be arriving
		{"\x1b[", Invalid, 2},

		// Recognized sequences keep their existing parsing
		{"\x1b[1;5A", CtrlUp, 6},
		{"\x1b[3;5~", CtrlDelete, 6},
		{"\x1b[2~", Insert, 4},
		{"\x1b[200~", BracketedPasteBegin, 6},
		{"\x1b[Z", ShiftTab, 3},
		{"\x1bOA", Up, 3},
		{"\x1b[12;34R", Invalid, 8},
		{"\x1b[?2004;2$y", Invalid, 11},
	} {
		r := &LightRenderer{buffer: []byte(c.sequence)}
		sz := 1
		event := r.escSequence(&sz)
		if event.Type != c.event {
			t.Errorf("escSequence(%q) = %s, want %s",
				c.sequence, event.Type.String(), c.event.String())
		}
		if sz != c.size {
			t.Errorf("escSequence(%q) consumed %d bytes, want %d", c.sequence, sz, c.size)
		}
	}
}

func TestStringEnd(t *testing.T) {
	for _, c := range []struct {
		buffer string
		want   int
	}{
		// Terminated
		{"\x1b]0;t\a", 6},
		{"\x1b]0;t\x1b\\", 7},
		{"\x1b_G\x1b\\", 5},
		{"\x1bP1$r\a\x1b\\", 8}, // BEL is payload in a DCS, ST ends it

		// BEL terminates an OSC only
		{"\x1bP\a", 0},
		{"\x1b_Gi=1\a", 0},

		// An ESC ends the string and introduces a sequence of its own
		{"\x1b]foo\x1bX\x1b\\", 5},
		{"\x1b]foo\x1bP", 5},
		{"\x1b]\x1b]", 2},

		// Nothing has ended it yet
		{"\x1b]0;t", 0},
		{"\x1b]0;t\x1b", 0}, // ST half arrived
		{"\x1b]foo\x1b", 0},
		{"\x1b]", 0},
		{"\x1b", 0}, // shorter than an introducer

	} {
		if got := stringEnd([]byte(c.buffer)); got != c.want {
			t.Errorf("stringEnd(%q) = %d, want %d", c.buffer, got, c.want)
		}
	}
}

// A dropped sequence followed by ALT-[ is not waiting for anything, so the
// bytes in between must not be held back until the next keystroke
func TestStillArriving(t *testing.T) {
	for _, c := range []struct {
		buffer string
		want   bool
	}{
		{"\x1b[97;5u\a\x1b[", false},
		{"\x1b[97;5u\x1b[", false},
		{"\x1b[3;\a\x1b[", false}, // the BEL ended it, the trailing \e[ is another one
		{"\x1b[1;\a", false},      // the BEL has already ended it
		{"\x1b[1;", true},
		{"\x1b[", true},
		{"\x1bO1", true},
		{"\x1b[<0;1", true},
	} {
		if got := stillArriving([]byte(c.buffer)); got != c.want {
			t.Errorf("stillArriving(%q) = %v, want %v", c.buffer, got, c.want)
		}
	}
}

func TestStringReply(t *testing.T) {
	for _, c := range []struct {
		buffer string
		want   bool
	}{
		{"\x1b]11;rgb:1/2/3\a", true},
		{"\x1b]0;title\a", true},
		{"\x1b]52;c;YWJj\x1b\\", true},
		{"\x1bP>|kitty(0.48.2)\x1b\\", true},
		{"\x1bP!|0\x1b\\", true},
		{"\x1bP1$r0m\x1b\\", true},
		{"\x1bP0+r\x1b\\", true},
		{"\x1b_Gi=1;OK\x1b\\", true},

		// What typed text after ALT-], ALT-P or ALT-_ looks like
		{"\x1b]\a", false},
		{"\x1b]a\a", false},
		{"\x1b]1\a", false},
		{"\x1b];\a", false},
		{"\x1bPx\x1b\\", false},
		{"\x1bP1\x1b\\", false},
		{"\x1b_ab\x1b\\", false},
		{"\x1b_G\x1b\\", false},
	} {
		if got := stringReply([]byte(c.buffer)); got != c.want {
			t.Errorf("stringReply(%q) = %v, want %v", c.buffer, got, c.want)
		}
	}
}

func TestCsiEnd(t *testing.T) {
	for _, c := range []struct {
		buffer string
		want   int
	}{
		{"\x1b[97;5u", 7},
		{"\x1b[A", 3},
		{"\x1b[<0;1;1M", 9},
		{"\x1b[?2004;2$y", 11},
		{"\x1b[7$x", 4}, // rxvt ends keys with $ after digits
		{"\x1b[23$", 5},
		{"\x1b[4;2$y", 7},   // but $ is intermediate elsewhere
		{"\x1b[97;5", 0},    // no final byte
		{"\x1b[", 0},        // no final byte
		{"\x1b[1\x01A", -1}, // malformed, do not frame it
		{"\x1b[\a", -1},
	} {
		if got := csiEnd([]byte(c.buffer)); got != c.want {
			t.Errorf("csiEnd(%q) = %d, want %d", c.buffer, got, c.want)
		}
	}
}
