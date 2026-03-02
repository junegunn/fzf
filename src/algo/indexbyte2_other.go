//go:build !arm64 && !amd64

package algo

import "bytes"

// indexByteTwo returns the index of the first occurrence of b1 or b2 in s,
// or -1 if neither is present.
func indexByteTwo(s []byte, b1, b2 byte) int {
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

// lastIndexByteTwo returns the index of the last occurrence of b1 or b2 in s,
// or -1 if neither is present.
func lastIndexByteTwo(s []byte, b1, b2 byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b1 || s[i] == b2 {
			return i
		}
	}
	return -1
}
