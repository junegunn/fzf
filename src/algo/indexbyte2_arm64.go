//go:build arm64

package algo

// indexByteTwo returns the index of the first occurrence of b1 or b2 in s,
// or -1 if neither is present. Implemented in assembly using ARM64 NEON
// to search for both bytes in a single pass.
//
//go:noescape
func indexByteTwo(s []byte, b1, b2 byte) int

// lastIndexByteTwo returns the index of the last occurrence of b1 or b2 in s,
// or -1 if neither is present. Implemented in assembly using ARM64 NEON,
// scanning backward.
//
//go:noescape
func lastIndexByteTwo(s []byte, b1, b2 byte) int
