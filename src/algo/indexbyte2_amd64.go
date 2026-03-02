//go:build amd64

package algo

var _useAVX2 bool

func init() {
	_useAVX2 = cpuHasAVX2()
}

//go:noescape
func cpuHasAVX2() bool

// indexByteTwo returns the index of the first occurrence of b1 or b2 in s,
// or -1 if neither is present. Uses AVX2 when available, SSE2 otherwise.
//
//go:noescape
func indexByteTwo(s []byte, b1, b2 byte) int

// lastIndexByteTwo returns the index of the last occurrence of b1 or b2 in s,
// or -1 if neither is present. Uses AVX2 when available, SSE2 otherwise.
//
//go:noescape
func lastIndexByteTwo(s []byte, b1, b2 byte) int
