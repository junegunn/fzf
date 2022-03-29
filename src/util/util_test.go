package util

import "testing"

func TestMax(t *testing.T) {
	if Max(-2, 5) != 5 {
		t.Error("Invalid result")
	}
}

func TestContrain(t *testing.T) {
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
