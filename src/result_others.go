//go:build !386 && !amd64

package fzf

func compareRanks(irank Result, jrank Result, tac bool) bool {
	for idx := 3; idx >= 0; idx-- {
		left := irank.points[idx]
		right := jrank.points[idx]
		if left < right {
			return true
		} else if left > right {
			return false
		}
	}
	return (irank.item.Index() <= jrank.item.Index()) != tac
}
