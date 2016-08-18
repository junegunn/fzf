package fzf

import (
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func TestStringPtr(t *testing.T) {
	orig := []byte("\x1b[34mfoo")
	text := []byte("\x1b[34mbar")
	item := Item{origText: &orig, text: util.ToChars(text)}
	if item.AsString(true) != "foo" || item.AsString(false) != string(orig) {
		t.Fail()
	}
	if item.AsString(true) != "foo" {
		t.Fail()
	}
	item.origText = nil
	if item.AsString(true) != string(text) || item.AsString(false) != string(text) {
		t.Fail()
	}
}
