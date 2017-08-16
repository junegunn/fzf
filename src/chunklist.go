package fzf

import "sync"

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

func (c *Chunk) push(trans ItemBuilder, data []byte) bool {
	if trans(&c.items[c.count], data) {
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

	ret := cl.lastChunk().push(cl.trans, data)
	cl.mutex.Unlock()
	return ret
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
