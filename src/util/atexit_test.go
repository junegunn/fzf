package util

import (
	"reflect"
	"testing"
)

func TestAtExit(t *testing.T) {
	want := []int{3, 2, 1, 0}
	var called []int
	for i := 0; i < 4; i++ {
		n := i
		AtExit(func() { called = append(called, n) })
	}
	RunAtExitFuncs()
	if !reflect.DeepEqual(called, want) {
		t.Errorf("AtExit: want call order: %v got: %v", want, called)
	}

	RunAtExitFuncs()
	if !reflect.DeepEqual(called, want) {
		t.Error("AtExit: should only call exit funcs once")
	}
}
