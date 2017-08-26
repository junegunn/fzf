// +build 386 amd64

package fzf

import "unsafe"

func compareRanks(irank Result, jrank Result, tac bool) bool {
	left := *(*uint64)(unsafe.Pointer(&irank.points[0]))
	right := *(*uint64)(unsafe.Pointer(&jrank.points[0]))
	if left < right {
		return true
	} else if left > right {
		return false
	}
	return (irank.item.Index() <= jrank.item.Index()) != tac
}
