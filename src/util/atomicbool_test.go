package util

import "testing"

func TestAtomicBool(t *testing.T) {
	if !NewAtomicBool(true).Get() || NewAtomicBool(false).Get() {
		t.Error("Invalid initial value")
	}

	ab := NewAtomicBool(true)
	if ab.Set(false) {
		t.Error("Invalid return value")
	}
	if ab.Get() {
		t.Error("Invalid state")
	}
}
