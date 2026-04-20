package fzf

import "sync"

// ChunkBitmap is a bitmap with one bit per item in a chunk.
type ChunkBitmap [chunkBitWords]uint64

// queryCache associates query strings to bitmaps of matching items
type queryCache map[string]ChunkBitmap

// ChunkCache associates Chunk and query string to bitmaps
type ChunkCache struct {
	mutex sync.Mutex
	cache map[*Chunk]*queryCache
}

// NewChunkCache returns a new ChunkCache
func NewChunkCache() *ChunkCache {
	return &ChunkCache{sync.Mutex{}, make(map[*Chunk]*queryCache)}
}

func (cc *ChunkCache) Clear() {
	cc.mutex.Lock()
	cc.cache = make(map[*Chunk]*queryCache)
	cc.mutex.Unlock()
}

func (cc *ChunkCache) retire(chunk ...*Chunk) {
	cc.mutex.Lock()
	for _, c := range chunk {
		delete(cc.cache, c)
	}
	cc.mutex.Unlock()
}

// Add stores the bitmap for the given chunk and key
func (cc *ChunkCache) Add(chunk *Chunk, key string, bitmap ChunkBitmap, matchCount int) {
	if len(key) == 0 || !chunk.IsFull() || matchCount > queryCacheMax {
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if !ok {
		cc.cache[chunk] = &queryCache{}
		qc = cc.cache[chunk]
	}
	(*qc)[key] = bitmap
}

// Lookup returns the bitmap for the exact key
func (cc *ChunkCache) Lookup(chunk *Chunk, key string) *ChunkBitmap {
	if len(key) == 0 || !chunk.IsFull() {
		return nil
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if ok {
		if bm, ok := (*qc)[key]; ok {
			return &bm
		}
	}
	return nil
}

// Search finds the bitmap for the longest prefix or suffix of the key
func (cc *ChunkCache) Search(chunk *Chunk, key string) *ChunkBitmap {
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
			if bm, found := (*qc)[substr]; found {
				return &bm
			}
		}
	}
	return nil
}
