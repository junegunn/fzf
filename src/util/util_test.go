package util

import "testing"

func TestMax(t *testing.T) {
	if Max(-2, 5, 1, 4, 3) != 5 {
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

func TestTrimLen(t *testing.T) {
	check := func(str string, exp int) {
		trimmed := TrimLen([]rune(str))
		if trimmed != exp {
			t.Errorf("Invalid TrimLen result for '%s': %d (expected %d)",
				str, trimmed, exp)
		}
	}
	check("hello", 5)
	check("hello ", 5)
	check("hello  ", 5)
	check(" hello", 5)
	check("  hello", 5)
	check(" hello ", 5)
	check("  hello  ", 5)
	check("h   o", 5)
	check("  h   o  ", 5)
	check("         ", 0)
}
