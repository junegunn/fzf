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
