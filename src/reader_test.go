package fzf

import "testing"

func TestReadFromCommand(t *testing.T) {
	strs := []string{}
	eb := NewEventBox()
	reader := Reader{
		pusher:   func(s string) { strs = append(strs, s) },
		eventBox: eb}

	// Check EventBox
	if eb.Peak(EVT_READ_NEW) {
		t.Error("EVT_READ_NEW should not be set yet")
	}

	// Normal command
	reader.readFromCommand(`echo abc && echo def`)
	if len(strs) != 2 || strs[0] != "abc" || strs[1] != "def" {
		t.Errorf("%s", strs)
	}

	// Check EventBox again
	if !eb.Peak(EVT_READ_NEW) {
		t.Error("EVT_READ_NEW should be set yet")
	}

	// Wait should return immediately
	eb.Wait(func(events *Events) {
		if _, found := (*events)[EVT_READ_NEW]; !found {
			t.Errorf("%s", events)
		}
		events.Clear()
	})

	// EventBox is cleared
	if eb.Peak(EVT_READ_NEW) {
		t.Error("EVT_READ_NEW should not be set yet")
	}

	// Failing command
	reader.readFromCommand(`no-such-command`)
	strs = []string{}
	if len(strs) > 0 {
		t.Errorf("%s", strs)
	}

	// Check EventBox again
	if eb.Peak(EVT_READ_NEW) {
		t.Error("Command failed. EVT_READ_NEW should be set")
	}
}
