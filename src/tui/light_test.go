package tui

import (
	"fmt"
	"os"
	"testing"
	"unicode"
)

func TestLightRenderer(t *testing.T) {
	tty_file, _ := os.Open("")
	renderer, _ := NewLightRenderer(
		"", tty_file, &ColorTheme{}, true, false, 0, false, true,
		func(h int) int { return h })

	light_renderer := renderer.(*LightRenderer)

	go func() {
		for {
			light_renderer.mutex.Lock()
			ready := light_renderer.cancel != nil
			light_renderer.mutex.Unlock()

			if ready {
				light_renderer.CancelGetChar()
				break
			}
		}
	}()
	event := light_renderer.GetChar(true)
	if event.Type != Invalid {
		t.Error("Not cancelled")
	}

	assertCharSequence := func(sequence string, name string) {
		bytes := []byte(sequence)
		light_renderer.buffer = bytes
		event := light_renderer.GetChar(true)
		if event.KeyName() != name {
			t.Errorf(
				"sequence: %q | %v | '%s' (%s) != %s",
				string(bytes), bytes,
				event.KeyName(), event.Type.String(), name)
		}
	}

	assertEscSequence := func(sequence string, name string) {
		bytes := []byte(sequence)
		light_renderer.buffer = bytes

		sz := 1
		event := light_renderer.escSequence(&sz)
		if fmt.Sprintf("!%s", event.Type.String()) == name {
			// this is fine
		} else if event.KeyName() != name {
			t.Errorf(
				"sequence: %q | %v | '%s' (%s) != %s",
				string(bytes), bytes,
				event.KeyName(), event.Type.String(), name)
		}
	}

	// invalid
	assertEscSequence("\x1b[<", "!Invalid")
	assertEscSequence("\x1b[1;1R", "!Invalid")
	assertEscSequence("\x1b[", "!Invalid")
	assertEscSequence("\x1b[1", "!Invalid")
	assertEscSequence("\x1b[3;3~1", "!Invalid")
	assertEscSequence("\x1b[13", "!Invalid")
	assertEscSequence("\x1b[1;3", "!Invalid")
	assertEscSequence("\x1b[1;10", "!Invalid")
	assertEscSequence("\x1b[220~", "!Invalid")
	assertEscSequence("\x1b[5;30~", "!Invalid")
	assertEscSequence("\x1b[6;30~", "!Invalid")

	// general
	for r := 'a'; r < 'z'; r++ {
		lower_r := fmt.Sprintf("%c", r)
		upper_r := fmt.Sprintf("%c", unicode.ToUpper(r))
		assertCharSequence(lower_r, lower_r)
		assertCharSequence(upper_r, upper_r)
	}

	assertCharSequence("\x01", "ctrl-a")
	assertCharSequence("\x02", "ctrl-b")
	assertCharSequence("\x03", "ctrl-c")
	assertCharSequence("\x04", "ctrl-d")
	assertCharSequence("\x05", "ctrl-e")
	assertCharSequence("\x06", "ctrl-f")
	assertCharSequence("\x07", "ctrl-g")
	// ctrl-h is the same as ctrl-backspace
	// ctrl-i is the same as tab
	assertCharSequence("\n", "ctrl-j")
	assertCharSequence("\x0b", "ctrl-k")
	assertCharSequence("\x0c", "ctrl-l")
	assertCharSequence("\r", "enter") // enter
	assertCharSequence("\x0e", "ctrl-n")
	assertCharSequence("\x0f", "ctrl-o")
	assertCharSequence("\x10", "ctrl-p")
	assertCharSequence("\x11", "ctrl-q")
	assertCharSequence("\x12", "ctrl-r")
	assertCharSequence("\x13", "ctrl-s")
	assertCharSequence("\x14", "ctrl-t")
	assertCharSequence("\x15", "ctrl-u")
	assertCharSequence("\x16", "ctrl-v")
	assertCharSequence("\x17", "ctrl-w")
	assertCharSequence("\x18", "ctrl-x")
	assertCharSequence("\x19", "ctrl-y")
	assertCharSequence("\x1a", "ctrl-z")

	assertCharSequence("\x00", "ctrl-space")
	assertCharSequence("\x1c", "ctrl-\\")
	assertCharSequence("\x1d", "ctrl-]")
	assertCharSequence("\x1e", "ctrl-^")
	assertCharSequence("\x1f", "ctrl-/")

	assertEscSequence("\x1ba", "alt-a")
	assertEscSequence("\x1bb", "alt-b")
	assertEscSequence("\x1bc", "alt-c")
	assertEscSequence("\x1bd", "alt-d")
	assertEscSequence("\x1be", "alt-e")
	assertEscSequence("\x1bf", "alt-f")
	assertEscSequence("\x1bg", "alt-g")
	assertEscSequence("\x1bh", "alt-h")
	assertEscSequence("\x1bi", "alt-i")
	assertEscSequence("\x1bj", "alt-j")
	assertEscSequence("\x1bk", "alt-k")
	assertEscSequence("\x1bl", "alt-l")
	assertEscSequence("\x1bm", "alt-m")
	assertEscSequence("\x1bn", "alt-n")
	assertEscSequence("\x1bo", "alt-o")
	assertEscSequence("\x1bp", "alt-p")
	assertEscSequence("\x1bq", "alt-q")
	assertEscSequence("\x1br", "alt-r")
	assertEscSequence("\x1bs", "alt-s")
	assertEscSequence("\x1bt", "alt-t")
	assertEscSequence("\x1bu", "alt-u")
	assertEscSequence("\x1bv", "alt-v")
	assertEscSequence("\x1bw", "alt-w")
	assertEscSequence("\x1bx", "alt-x")
	assertEscSequence("\x1by", "alt-y")
	assertEscSequence("\x1bz", "alt-z")

	assertEscSequence("\x1bOP", "f1")
	assertEscSequence("\x1bOQ", "f2")
	assertEscSequence("\x1bOR", "f3")
	assertEscSequence("\x1bOS", "f4")
	assertEscSequence("\x1b[15~", "f5")
	assertEscSequence("\x1b[17~", "f6")
	assertEscSequence("\x1b[18~", "f7")
	assertEscSequence("\x1b[19~", "f8")
	assertEscSequence("\x1b[20~", "f9")
	assertEscSequence("\x1b[21~", "f10")
	assertEscSequence("\x1b[23~", "f11")
	assertEscSequence("\x1b[24~", "f12")

	assertEscSequence("\x1b", "esc")
	assertCharSequence("\t", "tab")
	assertEscSequence("\x1b[Z", "shift-tab")

	assertCharSequence("\x7f", "backspace")
	assertEscSequence("\x1b\x7f", "alt-backspace")
	assertCharSequence("\b", "ctrl-backspace")
	assertEscSequence("\x1b\b", "ctrl-alt-backspace")

	assertEscSequence("\x1b[A", "up")
	assertEscSequence("\x1b[B", "down")
	assertEscSequence("\x1b[C", "right")
	assertEscSequence("\x1b[D", "left")
	assertEscSequence("\x1b[H", "home")
	assertEscSequence("\x1b[F", "end")
	assertEscSequence("\x1b[2~", "insert")
	assertEscSequence("\x1b[3~", "delete")
	assertEscSequence("\x1b[5~", "page-up")
	assertEscSequence("\x1b[6~", "page-down")
	assertEscSequence("\x1b[7~", "home")
	assertEscSequence("\x1b[8~", "end")

	assertEscSequence("\x1b[1;2A", "shift-up")
	assertEscSequence("\x1b[1;2B", "shift-down")
	assertEscSequence("\x1b[1;2C", "shift-right")
	assertEscSequence("\x1b[1;2D", "shift-left")
	assertEscSequence("\x1b[1;2H", "shift-home")
	assertEscSequence("\x1b[1;2F", "shift-end")
	assertEscSequence("\x1b[3;2~", "shift-delete")
	assertEscSequence("\x1b[5;2~", "shift-page-up")
	assertEscSequence("\x1b[6;2~", "shift-page-down")

	assertEscSequence("\x1b\x1b", "esc")
	assertEscSequence("\x1b\x1b[A", "alt-up")
	assertEscSequence("\x1b\x1b[B", "alt-down")
	assertEscSequence("\x1b\x1b[C", "alt-right")
	assertEscSequence("\x1b\x1b[D", "alt-left")

	assertEscSequence("\x1b[1;3A", "alt-up")
	assertEscSequence("\x1b[1;3B", "alt-down")
	assertEscSequence("\x1b[1;3C", "alt-right")
	assertEscSequence("\x1b[1;3D", "alt-left")
	assertEscSequence("\x1b[1;3H", "alt-home")
	assertEscSequence("\x1b[1;3F", "alt-end")
	assertEscSequence("\x1b[3;3~", "alt-delete")
	assertEscSequence("\x1b[5;3~", "alt-page-up")
	assertEscSequence("\x1b[6;3~", "alt-page-down")

	assertEscSequence("\x1b[1;4A", "alt-shift-up")
	assertEscSequence("\x1b[1;4B", "alt-shift-down")
	assertEscSequence("\x1b[1;4C", "alt-shift-right")
	assertEscSequence("\x1b[1;4D", "alt-shift-left")
	assertEscSequence("\x1b[1;4H", "alt-shift-home")
	assertEscSequence("\x1b[1;4F", "alt-shift-end")
	assertEscSequence("\x1b[3;4~", "alt-shift-delete")
	assertEscSequence("\x1b[5;4~", "alt-shift-page-up")
	assertEscSequence("\x1b[6;4~", "alt-shift-page-down")

	assertEscSequence("\x1b[1;5A", "ctrl-up")
	assertEscSequence("\x1b[1;5B", "ctrl-down")
	assertEscSequence("\x1b[1;5C", "ctrl-right")
	assertEscSequence("\x1b[1;5D", "ctrl-left")
	assertEscSequence("\x1b[1;5H", "ctrl-home")
	assertEscSequence("\x1b[1;5F", "ctrl-end")
	assertEscSequence("\x1b[3;5~", "ctrl-delete")
	assertEscSequence("\x1b[5;5~", "ctrl-page-up")
	assertEscSequence("\x1b[6;5~", "ctrl-page-down")

	assertEscSequence("\x1b[1;7A", "ctrl-alt-up")
	assertEscSequence("\x1b[1;7B", "ctrl-alt-down")
	assertEscSequence("\x1b[1;7C", "ctrl-alt-right")
	assertEscSequence("\x1b[1;7D", "ctrl-alt-left")
	assertEscSequence("\x1b[1;7H", "ctrl-alt-home")
	assertEscSequence("\x1b[1;7F", "ctrl-alt-end")
	assertEscSequence("\x1b[3;7~", "ctrl-alt-delete")
	assertEscSequence("\x1b[5;7~", "ctrl-alt-page-up")
	assertEscSequence("\x1b[6;7~", "ctrl-alt-page-down")

	assertEscSequence("\x1b[1;6A", "ctrl-shift-up")
	assertEscSequence("\x1b[1;6B", "ctrl-shift-down")
	assertEscSequence("\x1b[1;6C", "ctrl-shift-right")
	assertEscSequence("\x1b[1;6D", "ctrl-shift-left")
	assertEscSequence("\x1b[1;6H", "ctrl-shift-home")
	assertEscSequence("\x1b[1;6F", "ctrl-shift-end")
	assertEscSequence("\x1b[3;6~", "ctrl-shift-delete")
	assertEscSequence("\x1b[5;6~", "ctrl-shift-page-up")
	assertEscSequence("\x1b[6;6~", "ctrl-shift-page-down")

	assertEscSequence("\x1b[1;8A", "ctrl-alt-shift-up")
	assertEscSequence("\x1b[1;8B", "ctrl-alt-shift-down")
	assertEscSequence("\x1b[1;8C", "ctrl-alt-shift-right")
	assertEscSequence("\x1b[1;8D", "ctrl-alt-shift-left")
	assertEscSequence("\x1b[1;8H", "ctrl-alt-shift-home")
	assertEscSequence("\x1b[1;8F", "ctrl-alt-shift-end")
	assertEscSequence("\x1b[3;8~", "ctrl-alt-shift-delete")
	assertEscSequence("\x1b[5;8~", "ctrl-alt-shift-page-up")
	assertEscSequence("\x1b[6;8~", "ctrl-alt-shift-page-down")

	// xterm meta & mac
	assertEscSequence("\x1b[1;9A", "alt-up")
	assertEscSequence("\x1b[1;9B", "alt-down")
	assertEscSequence("\x1b[1;9C", "alt-right")
	assertEscSequence("\x1b[1;9D", "alt-left")
	assertEscSequence("\x1b[1;9H", "alt-home")
	assertEscSequence("\x1b[1;9F", "alt-end")
	assertEscSequence("\x1b[3;9~", "alt-delete")
	assertEscSequence("\x1b[5;9~", "alt-page-up")
	assertEscSequence("\x1b[6;9~", "alt-page-down")

	assertEscSequence("\x1b[1;10A", "alt-shift-up")
	assertEscSequence("\x1b[1;10B", "alt-shift-down")
	assertEscSequence("\x1b[1;10C", "alt-shift-right")
	assertEscSequence("\x1b[1;10D", "alt-shift-left")
	assertEscSequence("\x1b[1;10H", "alt-shift-home")
	assertEscSequence("\x1b[1;10F", "alt-shift-end")
	assertEscSequence("\x1b[3;10~", "alt-shift-delete")
	assertEscSequence("\x1b[5;10~", "alt-shift-page-up")
	assertEscSequence("\x1b[6;10~", "alt-shift-page-down")

	assertEscSequence("\x1b[1;11A", "alt-up")
	assertEscSequence("\x1b[1;11B", "alt-down")
	assertEscSequence("\x1b[1;11C", "alt-right")
	assertEscSequence("\x1b[1;11D", "alt-left")
	assertEscSequence("\x1b[1;11H", "alt-home")
	assertEscSequence("\x1b[1;11F", "alt-end")
	assertEscSequence("\x1b[3;11~", "alt-delete")
	assertEscSequence("\x1b[5;11~", "alt-page-up")
	assertEscSequence("\x1b[6;11~", "alt-page-down")

	assertEscSequence("\x1b[1;12A", "alt-shift-up")
	assertEscSequence("\x1b[1;12B", "alt-shift-down")
	assertEscSequence("\x1b[1;12C", "alt-shift-right")
	assertEscSequence("\x1b[1;12D", "alt-shift-left")
	assertEscSequence("\x1b[1;12H", "alt-shift-home")
	assertEscSequence("\x1b[1;12F", "alt-shift-end")
	assertEscSequence("\x1b[3;12~", "alt-shift-delete")
	assertEscSequence("\x1b[5;12~", "alt-shift-page-up")
	assertEscSequence("\x1b[6;12~", "alt-shift-page-down")

	assertEscSequence("\x1b[1;13A", "ctrl-alt-up")
	assertEscSequence("\x1b[1;13B", "ctrl-alt-down")
	assertEscSequence("\x1b[1;13C", "ctrl-alt-right")
	assertEscSequence("\x1b[1;13D", "ctrl-alt-left")
	assertEscSequence("\x1b[1;13H", "ctrl-alt-home")
	assertEscSequence("\x1b[1;13F", "ctrl-alt-end")
	assertEscSequence("\x1b[3;13~", "ctrl-alt-delete")
	assertEscSequence("\x1b[5;13~", "ctrl-alt-page-up")
	assertEscSequence("\x1b[6;13~", "ctrl-alt-page-down")

	assertEscSequence("\x1b[1;14A", "ctrl-alt-shift-up")
	assertEscSequence("\x1b[1;14B", "ctrl-alt-shift-down")
	assertEscSequence("\x1b[1;14C", "ctrl-alt-shift-right")
	assertEscSequence("\x1b[1;14D", "ctrl-alt-shift-left")
	assertEscSequence("\x1b[1;14H", "ctrl-alt-shift-home")
	assertEscSequence("\x1b[1;14F", "ctrl-alt-shift-end")
	assertEscSequence("\x1b[3;14~", "ctrl-alt-shift-delete")
	assertEscSequence("\x1b[5;14~", "ctrl-alt-shift-page-up")
	assertEscSequence("\x1b[6;14~", "ctrl-alt-shift-page-down")

	assertEscSequence("\x1b[1;15A", "ctrl-alt-up")
	assertEscSequence("\x1b[1;15B", "ctrl-alt-down")
	assertEscSequence("\x1b[1;15C", "ctrl-alt-right")
	assertEscSequence("\x1b[1;15D", "ctrl-alt-left")
	assertEscSequence("\x1b[1;15H", "ctrl-alt-home")
	assertEscSequence("\x1b[1;15F", "ctrl-alt-end")
	assertEscSequence("\x1b[3;15~", "ctrl-alt-delete")
	assertEscSequence("\x1b[5;15~", "ctrl-alt-page-up")
	assertEscSequence("\x1b[6;15~", "ctrl-alt-page-down")

	assertEscSequence("\x1b[1;16A", "ctrl-alt-shift-up")
	assertEscSequence("\x1b[1;16B", "ctrl-alt-shift-down")
	assertEscSequence("\x1b[1;16C", "ctrl-alt-shift-right")
	assertEscSequence("\x1b[1;16D", "ctrl-alt-shift-left")
	assertEscSequence("\x1b[1;16H", "ctrl-alt-shift-home")
	assertEscSequence("\x1b[1;16F", "ctrl-alt-shift-end")
	assertEscSequence("\x1b[3;16~", "ctrl-alt-shift-delete")
	assertEscSequence("\x1b[5;16~", "ctrl-alt-shift-page-up")
	assertEscSequence("\x1b[6;16~", "ctrl-alt-shift-page-down")

	// tmux & emacs
	assertEscSequence("\x1bOA", "up")
	assertEscSequence("\x1bOB", "down")
	assertEscSequence("\x1bOC", "right")
	assertEscSequence("\x1bOD", "left")
	assertEscSequence("\x1bOH", "home")
	assertEscSequence("\x1bOF", "end")

	// rrvt
	assertEscSequence("\x1b[1~", "home")
	assertEscSequence("\x1b[4~", "end")
	assertEscSequence("\x1b[11~", "f1")
	assertEscSequence("\x1b[12~", "f2")
	assertEscSequence("\x1b[13~", "f3")
	assertEscSequence("\x1b[14~", "f4")

}
