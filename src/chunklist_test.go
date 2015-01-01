package fzf

import (
	"fmt"
	"testing"
)

func TestChunkList(t *testing.T) {
	cl := NewChunkList(func(s *string, i int) *Item {
		return &Item{text: s, index: i * 2}
	})

	// Snapshot
	snapshot := cl.Snapshot()
	if len(snapshot) > 0 {
		t.Error("Snapshot should be empty now")
	}

	// Add some data
	cl.Push("hello")
	cl.Push("world")

	// Previously created snapshot should remain the same
	if len(snapshot) > 0 {
		t.Error("Snapshot should not have changed")
	}

	// But the new snapshot should contain the added items
	snapshot = cl.Snapshot()
	if len(snapshot) != 1 {
		t.Error("Snapshot should not be empty now")
	}

	// Check the content of the ChunkList
	chunk1 := snapshot[0]
	if len(*chunk1) != 2 {
		t.Error("Snapshot should contain only two items")
	}
	if *(*chunk1)[0].text != "hello" || (*chunk1)[0].index != 0 ||
		*(*chunk1)[1].text != "world" || (*chunk1)[1].index != 2 {
		t.Error("Invalid data")
	}
	if chunk1.IsFull() {
		t.Error("Chunk should not have been marked full yet")
	}

	// Add more data
	for i := 0; i < CHUNK_SIZE*2; i++ {
		cl.Push(fmt.Sprintf("item %d", i))
	}

	// Previous snapshot should remain the same
	if len(snapshot) != 1 {
		t.Error("Snapshot should stay the same")
	}

	// New snapshot
	snapshot = cl.Snapshot()
	if len(snapshot) != 3 || !snapshot[0].IsFull() ||
		!snapshot[1].IsFull() || snapshot[2].IsFull() {
		t.Error("Expected two full chunks and one more chunk")
	}
	if len(*snapshot[2]) != 2 {
		t.Error("Unexpected number of items")
	}
}
