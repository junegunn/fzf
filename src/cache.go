package fzf

import "sync"

// queryCache associates strings to lists of items
type queryCache map[string][]*Item

// ChunkCache associates Chunk and query string to lists of items
type ChunkCache struct {
	mutex sync.Mutex
	cache map[*Chunk]*queryCache
}

// NewChunkCache returns a new ChunkCache
func NewChunkCache() ChunkCache {
	return ChunkCache{sync.Mutex{}, make(map[*Chunk]*queryCache)}
}

// Add adds the list to the cache
func (cc *ChunkCache) Add(chunk *Chunk, key string, list []*Item) {
	if len(key) == 0 || !chunk.IsFull() || len(list) > queryCacheMax {
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if !ok {
		cc.cache[chunk] = &queryCache{}
		qc = cc.cache[chunk]
	}
	(*qc)[key] = list
}

// Find is called to lookup ChunkCache
func (cc *ChunkCache) Find(chunk *Chunk, key string) ([]*Item, bool) {
	if len(key) == 0 || !chunk.IsFull() {
		return nil, false
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if ok {
		list, ok := (*qc)[key]
		if ok {
			return list, true
		}
	}
	return nil, false
}
