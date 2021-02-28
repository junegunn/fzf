package fzf

import "testing"

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache()
	chunk1p := &Chunk{}
	chunk2p := &Chunk{count: chunkSize}
	items1 := []Result{{}}
	items2 := []Result{{}, {}}
	cache.Add(chunk1p, "foo", items1)
	cache.Add(chunk2p, "foo", items1)
	cache.Add(chunk2p, "bar", items2)

	{ // chunk1 is not full
		cached := cache.Lookup(chunk1p, "foo")
		if cached != nil {
			t.Error("Cached disabled for non-empty chunks", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "foo")
		if cached == nil || len(cached) != 1 {
			t.Error("Expected 1 item cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "bar")
		if cached == nil || len(cached) != 2 {
			t.Error("Expected 2 items cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk1p, "foobar")
		if cached != nil {
			t.Error("Expected 0 item cached", cached)
		}
	}
}
