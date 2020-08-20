package fzf

import (
	"bytes"
	"sync"
)

// Chunk is a list of Items whose size has the upper limit of chunkSize
type Chunk struct {
	items [chunkSize]Item
	count int
}

// ItemBuilder is a closure type that builds Item object from byte array
type ItemBuilder func(*Item, []byte) bool

// ChunkList is a list of Chunks
type ChunkList struct {
	chunks []*Chunk
	mutex  sync.Mutex
	trans  ItemBuilder
}

// NewChunkList returns a new ChunkList
func NewChunkList(trans ItemBuilder) *ChunkList {
	return &ChunkList{
		chunks: []*Chunk{},
		mutex:  sync.Mutex{},
		trans:  trans}
}

func (c *Chunk) push(duplicate func(line []byte) bool, trans ItemBuilder, data []byte) bool {
	item := &c.items[c.count]
	if trans(item, data) {
		bts := item.text.Bytes()

		// First check this chunk for duplicate items.
		for i, it := range c.items {
			if i == c.count {
				break
			}
			if bytes.Equal(bts, it.text.Bytes()) {
				return true
			}
		}

		// Second check all chunks for duplicate items.
		if duplicate(bts) {
			return true
		}

		// Only increment counter if we want to keep this item, it is not a duplicate.
		c.count++
		return true
	}
	return false
}

// IsFull returns true if the Chunk is full
func (c *Chunk) IsFull() bool {
	return c.count == chunkSize
}

func (cl *ChunkList) lastChunk() *Chunk {
	return cl.chunks[len(cl.chunks)-1]
}

// CountItems returns the total number of Items
func CountItems(cs []*Chunk) int {
	if len(cs) == 0 {
		return 0
	}
	return chunkSize*(len(cs)-1) + cs[len(cs)-1].count
}

// Push adds the item to the list
func (cl *ChunkList) Push(data []byte) bool {
	cl.mutex.Lock()

	if len(cl.chunks) == 0 || cl.lastChunk().IsFull() {
		cl.chunks = append(cl.chunks, &Chunk{})
	}

	ret := cl.lastChunk().push(func(bts []byte) bool {
		for i, chunk := range cl.chunks {
			// Break on the last item which is going to be the item being tested.
			if i == len(cl.chunks)-1 {
				break
			}

			for _, item := range chunk.items {
				if bytes.Equal(bts, item.text.Bytes()) {
					// Duplicate found.
					return true
				}
			}
		}
		// No dupliates found.
		return false
	}, cl.trans, data)
	cl.mutex.Unlock()
	return ret
}

// Clear clears the data
func (cl *ChunkList) Clear() {
	cl.mutex.Lock()
	cl.chunks = nil
	cl.mutex.Unlock()
}

// Snapshot returns immutable snapshot of the ChunkList
func (cl *ChunkList) Snapshot() ([]*Chunk, int) {
	cl.mutex.Lock()

	ret := make([]*Chunk, len(cl.chunks))
	copy(ret, cl.chunks)

	// Duplicate the last chunk
	if cnt := len(ret); cnt > 0 {
		newChunk := *ret[cnt-1]
		ret[cnt-1] = &newChunk
	}

	cl.mutex.Unlock()
	return ret, CountItems(ret)
}
