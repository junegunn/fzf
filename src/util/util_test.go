package util

import (
	"math"
	"strings"
	"testing"
)

func TestConstrain(t *testing.T) {
	if Constrain(-3, -1, 3) != -1 {
		t.Error("Expected", -1)
	}
	if Constrain(2, -1, 3) != 2 {
		t.Error("Expected", 2)
	}

	if Constrain(5, -1, 3) != 3 {
		t.Error("Expected", 3)
	}
}

func TestAsUint16(t *testing.T) {
	if AsUint16(5) != 5 {
		t.Error("Expected", 5)
	}
	if AsUint16(-10) != 0 {
		t.Error("Expected", 0)
	}
	if AsUint16(math.MaxUint16) != math.MaxUint16 {
		t.Error("Expected", math.MaxUint16)
	}
	if AsUint16(math.MinInt32) != 0 {
		t.Error("Expected", 0)
	}
	if AsUint16(math.MinInt16) != 0 {
		t.Error("Expected", 0)
	}
	if AsUint16(math.MaxUint16+1) != math.MaxUint16 {
		t.Error("Expected", math.MaxUint16)
	}
}

func TestOnce(t *testing.T) {
	o := Once(false)
	if o() {
		t.Error("Expected: false")
	}
	if !o() {
		t.Error("Expected: true")
	}
	if !o() {
		t.Error("Expected: true")
	}

	o = Once(true)
	if !o() {
		t.Error("Expected: true")
	}
	if o() {
		t.Error("Expected: false")
	}
	if o() {
		t.Error("Expected: false")
	}
}

func TestRunesWidth(t *testing.T) {
	for _, args := range [][]int{
		{100, 5, -1},
		{3, 4, 3},
		{0, 1, 0},
	} {
		width, overflowIdx := RunesWidth([]rune("hello"), 0, 0, args[0])
		if width != args[1] {
			t.Errorf("Expected width: %d, actual: %d", args[1], width)
		}
		if overflowIdx != args[2] {
			t.Errorf("Expected overflow index: %d, actual: %d", args[2], overflowIdx)
		}
	}
	for _, input := range []struct {
		s string
		w int
	}{
		{"▶", 1},
		{"▶️", 2},
	} {
		width, _ := RunesWidth([]rune(input.s), 0, 0, 100)
		if width != input.w {
			t.Errorf("Expected width of %s: %d, actual: %d", input.s, input.w, width)
		}
	}
}

func TestTruncate(t *testing.T) {
	truncated, width := Truncate("가나다라마", 7)
	if string(truncated) != "가나다" {
		t.Errorf("Expected: 가나다, actual: %s", string(truncated))
	}
	if width != 6 {
		t.Errorf("Expected: 6, actual: %d", width)
	}
}

func TestRepeatToFill(t *testing.T) {
	if RepeatToFill("abcde", 10, 50) != strings.Repeat("abcde", 5) {
		t.Error("Expected:", strings.Repeat("abcde", 5))
	}
	if RepeatToFill("abcde", 10, 42) != strings.Repeat("abcde", 4)+"abcde"[:2] {
		t.Error("Expected:", strings.Repeat("abcde", 4)+"abcde"[:2])
	}
}

func TestStringWidth(t *testing.T) {
	w := StringWidth("─")
	if w != 1 {
		t.Errorf("Expected: %d, Actual: %d", 1, w)
	}
}

func TestCompareVersions(t *testing.T) {
	assert := func(a, b string, expected int) {
		if result := CompareVersions(a, b); result != expected {
			t.Errorf("Expected: %d, Actual: %d", expected, result)
		}
	}

	assert("2", "1", 1)
	assert("2", "2", 0)
	assert("2", "10", -1)

	assert("2.1", "2.2", -1)
	assert("2.1", "2.1.1", -1)

	assert("1.2.3", "1.2.2", 1)
	assert("1.2.3", "1.2.3", 0)
	assert("1.2.3", "1.2.3.0", 0)
	assert("1.2.3", "1.2.4", -1)

	// Different number of parts
	assert("1.0.0", "1", 0)
	assert("1.0.0", "1.0", 0)
	assert("1.0.0", "1.0.0", 0)
	assert("1.0", "1.0.0", 0)
	assert("1", "1.0.0", 0)
	assert("1.0.0", "1.0.0.1", -1)
	assert("1.0.0.1.0", "1.0.0.1", 0)

	assert("", "3.4.5", -1)
}
