package fzf

import "sync"

// Chunk is a list of Item pointers whose size has the upper limit of chunkSize
type Chunk []*Item // >>> []Item

// ItemBuilder is a closure type that builds Item object from a pointer to a
// string and an integer
type ItemBuilder func([]byte, int) *Item

// ChunkList is a list of Chunks
type ChunkList struct {
	chunks []*Chunk
	count  int
	mutex  sync.Mutex
	trans  ItemBuilder
}

// NewChunkList returns a new ChunkList
func NewChunkList(trans ItemBuilder) *ChunkList {
	return &ChunkList{
		chunks: []*Chunk{},
		count:  0,
		mutex:  sync.Mutex{},
		trans:  trans}
}

func (c *Chunk) push(trans ItemBuilder, data []byte, index int) bool {
	item := trans(data, index)
	if item != nil {
		*c = append(*c, item)
		return true
	}
	return false
}

// IsFull returns true if the Chunk is full
func (c *Chunk) IsFull() bool {
	return len(*c) == chunkSize
}

func (cl *ChunkList) lastChunk() *Chunk {
	return cl.chunks[len(cl.chunks)-1]
}

// CountItems returns the total number of Items
func CountItems(cs []*Chunk) int {
	if len(cs) == 0 {
		return 0
	}
	return chunkSize*(len(cs)-1) + len(*(cs[len(cs)-1]))
}

// Push adds the item to the list
func (cl *ChunkList) Push(data []byte) bool {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	if len(cl.chunks) == 0 || cl.lastChunk().IsFull() {
		newChunk := Chunk(make([]*Item, 0, chunkSize))
		cl.chunks = append(cl.chunks, &newChunk)
	}

	if cl.lastChunk().push(cl.trans, data, cl.count) {
		cl.count++
		return true
	}
	return false
}

// Snapshot returns immutable snapshot of the ChunkList
func (cl *ChunkList) Snapshot() ([]*Chunk, int) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ret := make([]*Chunk, len(cl.chunks))
	copy(ret, cl.chunks)

	// Duplicate the last chunk
	if cnt := len(ret); cnt > 0 {
		ret[cnt-1] = ret[cnt-1].dupe()
	}
	return ret, cl.count
}

func (c *Chunk) dupe() *Chunk {
	newChunk := make(Chunk, len(*c))
	for idx, ptr := range *c {
		newChunk[idx] = ptr
	}
	return &newChunk
}
