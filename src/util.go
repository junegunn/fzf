package fzf

import "time"

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

func DurWithin(
	val time.Duration, min time.Duration, max time.Duration) time.Duration {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
