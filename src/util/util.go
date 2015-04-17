package util

// #include <unistd.h>
import "C"

import (
	"os"
	"time"
)

// Max returns the largest integer
func Max(first int, items ...int) int {
	max := first
	for _, item := range items {
		if item > max {
			max = item
		}
	}
	return max
}

// Max32 returns the smallest 32-bit integer
func Min32(first int32, second int32) int32 {
	if first <= second {
		return first
	}
	return second
}

// Max32 returns the largest 32-bit integer
func Max32(first int32, second int32) int32 {
	if first > second {
		return first
	}
	return second
}

// Constrain32 limits the given 32-bit integer with the upper and lower bounds
func Constrain32(val int32, min int32, max int32) int32 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// Constrain limits the given integer with the upper and lower bounds
func Constrain(val int, min int, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// DurWithin limits the given time.Duration with the upper and lower bounds
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

func Between(val int, min int, max int) bool {
	return val >= min && val <= max
}

// IsTty returns true is stdin is a terminal
func IsTty() bool {
	return int(C.isatty(C.int(os.Stdin.Fd()))) != 0
}

func TrimRight(runes *[]rune) []rune {
	var i int
	for i = len(*runes) - 1; i >= 0; i-- {
		char := (*runes)[i]
		if char != ' ' && char != '\t' {
			break
		}
	}
	return (*runes)[0 : i+1]
}
