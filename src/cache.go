package fzf

import "sync"

type QueryCache map[string][]*Item
type ChunkCache struct {
	mutex sync.Mutex
	cache map[*Chunk]*QueryCache
}

func NewChunkCache() ChunkCache {
	return ChunkCache{sync.Mutex{}, make(map[*Chunk]*QueryCache)}
}

func (cc *ChunkCache) Add(chunk *Chunk, key string, list []*Item) {
	if len(key) == 0 || !chunk.IsFull() {
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if !ok {
		cc.cache[chunk] = &QueryCache{}
		qc = cc.cache[chunk]
	}
	(*qc)[key] = list
}

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
