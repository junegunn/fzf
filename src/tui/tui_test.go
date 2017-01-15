package tui

import "testing"

func TestHexToColor(t *testing.T) {
	assert := func(expr string, r, g, b int) {
		color := HexToColor(expr)
		if !color.is24() ||
			int((color>>16)&0xff) != r ||
			int((color>>8)&0xff) != g ||
			int((color)&0xff) != b {
			t.Fail()
		}
	}

	assert("#ff0000", 255, 0, 0)
	assert("#010203", 1, 2, 3)
	assert("#102030", 16, 32, 48)
	assert("#ffffff", 255, 255, 255)
}
