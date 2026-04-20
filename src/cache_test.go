package fzf

import "testing"

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache()
	chunk1p := &Chunk{}
	chunk2p := &Chunk{count: chunkSize}
	bm1 := ChunkBitmap{1}
	bm2 := ChunkBitmap{1, 2}
	cache.Add(chunk1p, "foo", bm1, 1)
	cache.Add(chunk2p, "foo", bm1, 1)
	cache.Add(chunk2p, "bar", bm2, 2)

	{ // chunk1 is not full
		cached := cache.Lookup(chunk1p, "foo")
		if cached != nil {
			t.Error("Cached disabled for non-full chunks", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "foo")
		if cached == nil || cached[0] != 1 {
			t.Error("Expected bitmap cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "bar")
		if cached == nil || cached[1] != 2 {
			t.Error("Expected bitmap cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk1p, "foobar")
		if cached != nil {
			t.Error("Expected nil cached", cached)
		}
	}
}
