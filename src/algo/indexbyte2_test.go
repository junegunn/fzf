package algo

import (
	"bytes"
	"testing"
)

func TestIndexByteTwo(t *testing.T) {
	tests := []struct {
		name string
		s    string
		b1   byte
		b2   byte
		want int
	}{
		{"empty", "", 'a', 'b', -1},
		{"single_b1", "a", 'a', 'b', 0},
		{"single_b2", "b", 'a', 'b', 0},
		{"single_none", "c", 'a', 'b', -1},
		{"b1_first", "xaxb", 'a', 'b', 1},
		{"b2_first", "xbxa", 'a', 'b', 1},
		{"same_byte", "xxa", 'a', 'a', 2},
		{"at_end", "xxxxa", 'a', 'b', 4},
		{"not_found", "xxxxxxxx", 'a', 'b', -1},
		{"long_b1_at_3000", string(make([]byte, 3000)) + "a" + string(make([]byte, 1000)), 'a', 'b', 3000},
		{"long_b2_at_3000", string(make([]byte, 3000)) + "b" + string(make([]byte, 1000)), 'a', 'b', 3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexByteTwo([]byte(tt.s), tt.b1, tt.b2)
			if got != tt.want {
				t.Errorf("indexByteTwo(%q, %c, %c) = %d, want %d", tt.s[:min(len(tt.s), 40)], tt.b1, tt.b2, got, tt.want)
			}
		})
	}

	// Exhaustive test: compare against loop reference for various lengths,
	// including sizes around SIMD block boundaries (16, 32, 64).
	for n := 0; n <= 256; n++ {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte('c' + (i % 20))
		}
		// Test with match at every position
		for pos := 0; pos < n; pos++ {
			for _, b := range []byte{'A', 'B'} {
				data[pos] = b
				got := indexByteTwo(data, 'A', 'B')
				want := loopIndexByteTwo(data, 'A', 'B')
				if got != want {
					t.Fatalf("indexByteTwo(len=%d, match=%c@%d) = %d, want %d", n, b, pos, got, want)
				}
				data[pos] = byte('c' + (pos % 20))
			}
		}
		// Test with no match
		got := indexByteTwo(data, 'A', 'B')
		if got != -1 {
			t.Fatalf("indexByteTwo(len=%d, no match) = %d, want -1", n, got)
		}
		// Test with both bytes present
		if n >= 2 {
			data[n/3] = 'A'
			data[n*2/3] = 'B'
			got := indexByteTwo(data, 'A', 'B')
			want := loopIndexByteTwo(data, 'A', 'B')
			if got != want {
				t.Fatalf("indexByteTwo(len=%d, both@%d,%d) = %d, want %d", n, n/3, n*2/3, got, want)
			}
			data[n/3] = byte('c' + ((n / 3) % 20))
			data[n*2/3] = byte('c' + ((n * 2 / 3) % 20))
		}
	}
}

func TestLastIndexByteTwo(t *testing.T) {
	tests := []struct {
		name string
		s    string
		b1   byte
		b2   byte
		want int
	}{
		{"empty", "", 'a', 'b', -1},
		{"single_b1", "a", 'a', 'b', 0},
		{"single_b2", "b", 'a', 'b', 0},
		{"single_none", "c", 'a', 'b', -1},
		{"b1_last", "xbxa", 'a', 'b', 3},
		{"b2_last", "xaxb", 'a', 'b', 3},
		{"same_byte", "axx", 'a', 'a', 0},
		{"at_start", "axxxx", 'a', 'b', 0},
		{"both_present", "axbx", 'a', 'b', 2},
		{"not_found", "xxxxxxxx", 'a', 'b', -1},
		{"long_b1_at_3000", string(make([]byte, 3000)) + "a" + string(make([]byte, 1000)), 'a', 'b', 3000},
		{"long_b2_at_end", string(make([]byte, 4000)) + "b", 'a', 'b', 4000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastIndexByteTwo([]byte(tt.s), tt.b1, tt.b2)
			if got != tt.want {
				t.Errorf("lastIndexByteTwo(%q, %c, %c) = %d, want %d", tt.s[:min(len(tt.s), 40)], tt.b1, tt.b2, got, tt.want)
			}
		})
	}

	// Exhaustive test against loop reference
	for n := 0; n <= 256; n++ {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte('c' + (i % 20))
		}
		for pos := 0; pos < n; pos++ {
			for _, b := range []byte{'A', 'B'} {
				data[pos] = b
				got := lastIndexByteTwo(data, 'A', 'B')
				want := refLastIndexByteTwo(data, 'A', 'B')
				if got != want {
					t.Fatalf("lastIndexByteTwo(len=%d, match=%c@%d) = %d, want %d", n, b, pos, got, want)
				}
				data[pos] = byte('c' + (pos % 20))
			}
		}
		// No match
		got := lastIndexByteTwo(data, 'A', 'B')
		if got != -1 {
			t.Fatalf("lastIndexByteTwo(len=%d, no match) = %d, want -1", n, got)
		}
		// Both bytes present
		if n >= 2 {
			data[n/3] = 'A'
			data[n*2/3] = 'B'
			got := lastIndexByteTwo(data, 'A', 'B')
			want := refLastIndexByteTwo(data, 'A', 'B')
			if got != want {
				t.Fatalf("lastIndexByteTwo(len=%d, both@%d,%d) = %d, want %d", n, n/3, n*2/3, got, want)
			}
			data[n/3] = byte('c' + ((n / 3) % 20))
			data[n*2/3] = byte('c' + ((n * 2 / 3) % 20))
		}
	}
}

func FuzzIndexByteTwo(f *testing.F) {
	f.Add([]byte("hello world"), byte('o'), byte('l'))
	f.Add([]byte(""), byte('a'), byte('b'))
	f.Add([]byte("aaa"), byte('a'), byte('a'))
	f.Fuzz(func(t *testing.T, data []byte, b1, b2 byte) {
		got := indexByteTwo(data, b1, b2)
		want := loopIndexByteTwo(data, b1, b2)
		if got != want {
			t.Errorf("indexByteTwo(len=%d, b1=%d, b2=%d) = %d, want %d", len(data), b1, b2, got, want)
		}
	})
}

func FuzzLastIndexByteTwo(f *testing.F) {
	f.Add([]byte("hello world"), byte('o'), byte('l'))
	f.Add([]byte(""), byte('a'), byte('b'))
	f.Add([]byte("aaa"), byte('a'), byte('a'))
	f.Fuzz(func(t *testing.T, data []byte, b1, b2 byte) {
		got := lastIndexByteTwo(data, b1, b2)
		want := refLastIndexByteTwo(data, b1, b2)
		if got != want {
			t.Errorf("lastIndexByteTwo(len=%d, b1=%d, b2=%d) = %d, want %d", len(data), b1, b2, got, want)
		}
	})
}

// Reference implementations for correctness checking
func refIndexByteTwo(s []byte, b1, b2 byte) int {
	i1 := bytes.IndexByte(s, b1)
	if i1 == 0 {
		return 0
	}
	scope := s
	if i1 > 0 {
		scope = s[:i1]
	}
	if i2 := bytes.IndexByte(scope, b2); i2 >= 0 {
		return i2
	}
	return i1
}

func loopIndexByteTwo(s []byte, b1, b2 byte) int {
	for i, b := range s {
		if b == b1 || b == b2 {
			return i
		}
	}
	return -1
}

func refLastIndexByteTwo(s []byte, b1, b2 byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b1 || s[i] == b2 {
			return i
		}
	}
	return -1
}

func benchIndexByteTwo(b *testing.B, size int, pos int) {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte('a' + (i % 20))
	}
	data[pos] = 'Z'

	type impl struct {
		name string
		fn   func([]byte, byte, byte) int
	}
	impls := []impl{
		{"asm", indexByteTwo},
		{"2xIndexByte", refIndexByteTwo},
		{"loop", loopIndexByteTwo},
	}
	for _, im := range impls {
		b.Run(im.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				im.fn(data, 'Z', 'z')
			}
		})
	}
}

func benchLastIndexByteTwo(b *testing.B, size int, pos int) {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte('a' + (i % 20))
	}
	data[pos] = 'Z'

	type impl struct {
		name string
		fn   func([]byte, byte, byte) int
	}
	impls := []impl{
		{"asm", lastIndexByteTwo},
		{"loop", refLastIndexByteTwo},
	}
	for _, im := range impls {
		b.Run(im.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				im.fn(data, 'Z', 'z')
			}
		})
	}
}

func BenchmarkIndexByteTwo_10(b *testing.B)       { benchIndexByteTwo(b, 10, 8) }
func BenchmarkIndexByteTwo_100(b *testing.B)      { benchIndexByteTwo(b, 100, 80) }
func BenchmarkIndexByteTwo_1000(b *testing.B)     { benchIndexByteTwo(b, 1000, 800) }
func BenchmarkLastIndexByteTwo_10(b *testing.B)   { benchLastIndexByteTwo(b, 10, 2) }
func BenchmarkLastIndexByteTwo_100(b *testing.B)  { benchLastIndexByteTwo(b, 100, 20) }
func BenchmarkLastIndexByteTwo_1000(b *testing.B) { benchLastIndexByteTwo(b, 1000, 200) }
