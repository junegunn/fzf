package fzf

import (
	"fmt"
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func TestChunkList(t *testing.T) {
	// FIXME global
	sortCriteria = []criterion{byScore, byLength}

	cl := NewChunkList(func(s []byte, i int) *Item {
		return &Item{text: util.ToChars(s), index: int32(i * 2)}
	})

	// Snapshot
	snapshot, count := cl.Snapshot()
	if len(snapshot) > 0 || count > 0 {
		t.Error("Snapshot should be empty now")
	}

	// Add some data
	cl.Push([]byte("hello"))
	cl.Push([]byte("world"))

	// Previously created snapshot should remain the same
	if len(snapshot) > 0 {
		t.Error("Snapshot should not have changed")
	}

	// But the new snapshot should contain the added items
	snapshot, count = cl.Snapshot()
	if len(snapshot) != 1 && count != 2 {
		t.Error("Snapshot should not be empty now")
	}

	// Check the content of the ChunkList
	chunk1 := snapshot[0]
	if len(*chunk1) != 2 {
		t.Error("Snapshot should contain only two items")
	}
	if (*chunk1)[0].text.ToString() != "hello" || (*chunk1)[0].index != 0 ||
		(*chunk1)[1].text.ToString() != "world" || (*chunk1)[1].index != 2 {
		t.Error("Invalid data")
	}
	if chunk1.IsFull() {
		t.Error("Chunk should not have been marked full yet")
	}

	// Add more data
	for i := 0; i < chunkSize*2; i++ {
		cl.Push([]byte(fmt.Sprintf("item %d", i)))
	}

	// Previous snapshot should remain the same
	if len(snapshot) != 1 {
		t.Error("Snapshot should stay the same")
	}

	// New snapshot
	snapshot, count = cl.Snapshot()
	if len(snapshot) != 3 || !snapshot[0].IsFull() ||
		!snapshot[1].IsFull() || snapshot[2].IsFull() || count != chunkSize*2+2 {
		t.Error("Expected two full chunks and one more chunk")
	}
	if len(*snapshot[2]) != 2 {
		t.Error("Unexpected number of items")
	}

	cl.Push([]byte("hello"))
	cl.Push([]byte("world"))

	lastChunkCount := len(*snapshot[len(snapshot)-1])
	if lastChunkCount != 2 {
		t.Error("Unexpected number of items:", lastChunkCount)
	}
}
