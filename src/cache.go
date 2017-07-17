package fzf

import "sync"

// queryCache associates strings to lists of items
type queryCache map[string][]Result

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
func (cc *ChunkCache) Add(chunk *Chunk, key string, list []Result) {
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

// Lookup is called to lookup ChunkCache
func (cc *ChunkCache) Lookup(chunk *Chunk, key string) []Result {
	if len(key) == 0 || !chunk.IsFull() {
		return nil
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if ok {
		list, ok := (*qc)[key]
		if ok {
			return list
		}
	}
	return nil
}

func (cc *ChunkCache) Search(chunk *Chunk, key string) []Result {
	if len(key) == 0 || !chunk.IsFull() {
		return nil
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if !ok {
		return nil
	}

	for idx := 1; idx < len(key); idx++ {
		// [---------| ] | [ |---------]
		// [--------|  ] | [  |--------]
		// [-------|   ] | [   |-------]
		prefix := key[:len(key)-idx]
		suffix := key[idx:]
		for _, substr := range [2]string{prefix, suffix} {
			if cached, found := (*qc)[substr]; found {
				return cached
			}
		}
	}
	return nil
}
