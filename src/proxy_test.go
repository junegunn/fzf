package fzf

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestLabelUpdater(t *testing.T) {
	var mu sync.Mutex
	applied := []int{}
	// The goroutine can still be running after the test function returns, so
	// it records the label instead of reporting it, and -1 stands in for one
	// that could not be parsed
	updater := &labelUpdater{set: func(label string) {
		n, err := strconv.Atoi(label)
		if err != nil {
			n = -1
		}
		mu.Lock()
		applied = append(applied, n)
		mu.Unlock()
		// Long enough for the burst to pile up behind the first update
		time.Sleep(time.Millisecond)
	}}

	const last = 99
	for i := 0; i <= last; i++ {
		updater.update(strconv.Itoa(i))
	}

	// However many updates are dropped, the final one is always applied
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		done := len(applied) > 0 && applied[len(applied)-1] == last
		mu.Unlock()
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("last update was not applied: %v", applied)
		}
		time.Sleep(time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	// Updates are never applied out of order, so the border cannot be left
	// showing a label older than the last one requested
	for i, n := range applied {
		if i > 0 && n <= applied[i-1] {
			t.Errorf("applied out of order: %v", applied)
			break
		}
	}
	// A burst collapses instead of running one command per update. The
	// updates are queued in a tight loop while each one takes a millisecond
	// to apply, so all of them landing would mean no coalescing at all
	if len(applied) >= last+1 {
		t.Errorf("expected updates to be coalesced, applied all %d", len(applied))
	}
	t.Logf("applied %d of %d updates", len(applied), last+1)
}
