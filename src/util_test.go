package fzf

import "testing"

func TestMax(t *testing.T) {
	if Max(-2, 5, 1, 4, 3) != 5 {
		t.Error("Invalid result")
	}
}

func TestMin(t *testing.T) {
	if Min(2, -3) != -3 {
		t.Error("Invalid result")
	}
	if Min(-2, 5, 1, 4, 3) != -2 {
		t.Error("Invalid result")
	}
}
