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
	cache  *ChunkCache
}

// NewChunkList returns a new ChunkList
func NewChunkList(cache *ChunkCache, trans ItemBuilder) *ChunkList {
	return &ChunkList{
		chunks: []*Chunk{},
		mutex:  sync.Mutex{},
		trans:  trans,
		cache:  cache}
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
	if len(cs) == 1 {
		return cs[0].count
	}

	// First chunk might not be full due to --tail=N
	return cs[0].count + chunkSize*(len(cs)-2) + cs[len(cs)-1].count
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

// Clear clears the data
func (cl *ChunkList) Clear() {
	cl.mutex.Lock()
	cl.chunks = nil
	cl.mutex.Unlock()
}

// Snapshot returns immutable snapshot of the ChunkList
func (cl *ChunkList) Snapshot(tail int) ([]*Chunk, int, bool) {
	cl.mutex.Lock()

	changed := false
	if tail > 0 && CountItems(cl.chunks) > tail {
		changed = true
		// Find the number of chunks to keep
		numChunks := 0
		for left, i := tail, len(cl.chunks)-1; left > 0 && i >= 0; i-- {
			numChunks++
			left -= cl.chunks[i].count
		}

		// Copy the chunks to keep
		ret := make([]*Chunk, numChunks)
		minIndex := len(cl.chunks) - numChunks
		cl.cache.retire(cl.chunks[:minIndex]...)
		copy(ret, cl.chunks[minIndex:])

		for left, i := tail, len(ret)-1; i >= 0; i-- {
			chunk := ret[i]
			if chunk.count > left {
				newChunk := *chunk
				newChunk.count = left
				oldCount := chunk.count
				for i := 0; i < left; i++ {
					newChunk.items[i] = chunk.items[oldCount-left+i]
				}
				ret[i] = &newChunk
				cl.cache.retire(chunk)
				break
			}
			left -= chunk.count
		}
		cl.chunks = ret
	}

	ret := make([]*Chunk, len(cl.chunks))
	copy(ret, cl.chunks)

	// Duplicate the first and the last chunk
	if cnt := len(ret); cnt > 0 {
		if tail > 0 && cnt > 1 {
			newChunk := *ret[0]
			ret[0] = &newChunk
		}
		newChunk := *ret[cnt-1]
		ret[cnt-1] = &newChunk
	}

	cl.mutex.Unlock()
	return ret, CountItems(ret), changed
}
