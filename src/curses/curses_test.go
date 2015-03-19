package curses

import (
	"testing"
)

func TestPairFor(t *testing.T) {
	if PairFor(30, 50) != PairFor(30, 50) {
		t.Fail()
	}
	if PairFor(-1, 10) != PairFor(-1, 10) {
		t.Fail()
	}
}
