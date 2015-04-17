package fzf

import "testing"

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache()
	chunk2 := make(Chunk, chunkSize)
	chunk1p := &Chunk{}
	chunk2p := &chunk2
	items1 := []*Item{&Item{}}
	items2 := []*Item{&Item{}, &Item{}}
	cache.Add(chunk1p, "foo", items1)
	cache.Add(chunk2p, "foo", items1)
	cache.Add(chunk2p, "bar", items2)

	{ // chunk1 is not full
		cached, found := cache.Find(chunk1p, "foo")
		if found {
			t.Error("Cached disabled for non-empty chunks", found, cached)
		}
	}
	{
		cached, found := cache.Find(chunk2p, "foo")
		if !found || len(cached) != 1 {
			t.Error("Expected 1 item cached", found, cached)
		}
	}
	{
		cached, found := cache.Find(chunk2p, "bar")
		if !found || len(cached) != 2 {
			t.Error("Expected 2 items cached", found, cached)
		}
	}
	{
		cached, found := cache.Find(chunk1p, "foobar")
		if found {
			t.Error("Expected 0 item cached", found, cached)
		}
	}
}
