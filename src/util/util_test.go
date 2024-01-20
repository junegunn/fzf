package util

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestMax(t *testing.T) {
	if Max(10, 1) != 10 {
		t.Error("Expected", 10)
	}
	if Max(-2, 5) != 5 {
		t.Error("Expected", 5)
	}
}

func TestMax16(t *testing.T) {
	if Max16(10, 1) != 10 {
		t.Error("Expected", 10)
	}
	if Max16(-2, 5) != 5 {
		t.Error("Expected", 5)
	}
	if Max16(math.MaxInt16, 0) != math.MaxInt16 {
		t.Error("Expected", math.MaxInt16)
	}
	if Max16(0, math.MinInt16) != 0 {
		t.Error("Expected", 0)
	}
}

func TestMax32(t *testing.T) {
	if Max32(10, 1) != 10 {
		t.Error("Expected", 10)
	}
	if Max32(-2, 5) != 5 {
		t.Error("Expected", 5)
	}
	if Max32(math.MaxInt32, 0) != math.MaxInt32 {
		t.Error("Expected", math.MaxInt32)
	}
	if Max32(0, math.MinInt32) != 0 {
		t.Error("Expected", 0)
	}
}

func TestMin(t *testing.T) {
	if Min(10, 1) != 1 {
		t.Error("Expected", 1)
	}
	if Min(-2, 5) != -2 {
		t.Error("Expected", -2)
	}
}

func TestMin32(t *testing.T) {
	if Min32(10, 1) != 1 {
		t.Error("Expected", 1)
	}
	if Min32(-2, 5) != -2 {
		t.Error("Expected", -2)
	}
	if Min32(math.MaxInt32, 0) != 0 {
		t.Error("Expected", 0)
	}
	if Min32(0, math.MinInt32) != math.MinInt32 {
		t.Error("Expected", math.MinInt32)
	}
}

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

func TestConstrain32(t *testing.T) {
	if Constrain32(-3, -1, 3) != -1 {
		t.Error("Expected", -1)
	}
	if Constrain32(2, -1, 3) != 2 {
		t.Error("Expected", 2)
	}

	if Constrain32(5, -1, 3) != 3 {
		t.Error("Expected", 3)
	}
	if Constrain32(0, math.MinInt32, math.MaxInt32) != 0 {
		t.Error("Expected", 0)
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

func TestDurWithIn(t *testing.T) {
	if DurWithin(time.Duration(5), time.Duration(1), time.Duration(8)) != time.Duration(5) {
		t.Error("Expected", time.Duration(0))
	}
	if DurWithin(time.Duration(0)*time.Second, time.Second, time.Duration(3)*time.Second) != time.Second {
		t.Error("Expected", time.Second)
	}
	if DurWithin(time.Duration(10)*time.Second, time.Duration(0), time.Second) != time.Second {
		t.Error("Expected", time.Second)
	}
}

func TestOnce(t *testing.T) {
	o := Once(false)
	if o() {
		t.Error("Expected: false")
	}
	if o() {
		t.Error("Expected: false")
	}

	o = Once(true)
	if !o() {
		t.Error("Expected: true")
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
