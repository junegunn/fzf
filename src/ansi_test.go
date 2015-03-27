package fzf

import (
	"fmt"
	"testing"
)

func TestExtractColor(t *testing.T) {
	assert := func(offset ansiOffset, b int32, e int32, fg int, bg int, bold bool) {
		if offset.offset[0] != b || offset.offset[1] != e ||
			offset.color.fg != fg || offset.color.bg != bg || offset.color.bold != bold {
			t.Error(offset, b, e, fg, bg, bold)
		}
	}

	src := "hello world"
	clean := "\x1b[0m"
	check := func(assertion func(ansiOffsets []ansiOffset)) {
		output, ansiOffsets := extractColor(&src)
		if *output != "hello world" {
			t.Errorf("Invalid output: {}", output)
		}
		fmt.Println(src, ansiOffsets, clean)
		assertion(ansiOffsets)
	}

	check(func(offsets []ansiOffset) {
		if len(offsets) > 0 {
			t.Fail()
		}
	})

	src = "\x1b[0mhello world"
	check(func(offsets []ansiOffset) {
		if len(offsets) > 0 {
			t.Fail()
		}
	})

	src = "\x1b[1mhello world"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 1 {
			t.Fail()
		}
		assert(offsets[0], 0, 11, -1, -1, true)
	})

	src = "\x1b[1mhello \x1b[mworld"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 1 {
			t.Fail()
		}
		assert(offsets[0], 0, 6, -1, -1, true)
	})

	src = "\x1b[1mhello \x1b[Kworld"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 1 {
			t.Fail()
		}
		assert(offsets[0], 0, 11, -1, -1, true)
	})

	src = "hello \x1b[34;45;1mworld"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 1 {
			t.Fail()
		}
		assert(offsets[0], 6, 11, 4, 5, true)
	})

	src = "hello \x1b[34;45;1mwor\x1b[34;45;1mld"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 1 {
			t.Fail()
		}
		assert(offsets[0], 6, 11, 4, 5, true)
	})

	src = "hello \x1b[34;45;1mwor\x1b[0mld"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 1 {
			t.Fail()
		}
		assert(offsets[0], 6, 9, 4, 5, true)
	})

	src = "hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 3 {
			t.Fail()
		}
		assert(offsets[0], 6, 8, 4, 233, true)
		assert(offsets[1], 8, 9, 161, 233, true)
		assert(offsets[2], 10, 11, 161, -1, false)
	})

	// {38,48};5;{38,48}
	src = "hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md"
	check(func(offsets []ansiOffset) {
		if len(offsets) != 2 {
			t.Fail()
		}
		assert(offsets[0], 6, 9, 38, 48, true)
		assert(offsets[1], 9, 10, 48, 38, true)
	})
}
