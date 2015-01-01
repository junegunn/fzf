package fzf

func Max(first int, items ...int) int {
	max := first
	for _, item := range items {
		if item > max {
			max = item
		}
	}
	return max
}

func Min(first int, items ...int) int {
	min := first
	for _, item := range items {
		if item < min {
			min = item
		}
	}
	return min
}
