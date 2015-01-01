package fzf

import "sync"

const CHUNK_SIZE int = 100

type Chunk []*Item // >>> []Item

type Transformer func(*string, int) *Item

type ChunkList struct {
	chunks []*Chunk
	count  int
	mutex  sync.Mutex
	trans  Transformer
}

func NewChunkList(trans Transformer) *ChunkList {
	return &ChunkList{
		chunks: []*Chunk{},
		count:  0,
		mutex:  sync.Mutex{},
		trans:  trans}
}

func (c *Chunk) push(trans Transformer, data *string, index int) {
	*c = append(*c, trans(data, index))
}

func (c *Chunk) IsFull() bool {
	return len(*c) == CHUNK_SIZE
}

func (cl *ChunkList) lastChunk() *Chunk {
	return cl.chunks[len(cl.chunks)-1]
}

func CountItems(cs []*Chunk) int {
	if len(cs) == 0 {
		return 0
	}
	return CHUNK_SIZE*(len(cs)-1) + len(*(cs[len(cs)-1]))
}

func (cl *ChunkList) Count() int {
	return cl.count
}

func (cl *ChunkList) Chunks() []*Chunk {
	return cl.chunks
}

func (cl *ChunkList) Push(data string) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	if len(cl.chunks) == 0 || cl.lastChunk().IsFull() {
		newChunk := Chunk(make([]*Item, 0, CHUNK_SIZE))
		cl.chunks = append(cl.chunks, &newChunk)
	}

	cl.lastChunk().push(cl.trans, &data, cl.count)
	cl.count += 1
}

func (cl *ChunkList) Snapshot() []*Chunk {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	ret := make([]*Chunk, len(cl.chunks))
	copy(ret, cl.chunks)
	return ret
}
