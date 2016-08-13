package util

// #include <unistd.h>
import "C"

import (
	"os"
	"os/exec"
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

// Min returns the smallest integer
func Min(first int, second int) int {
	if first <= second {
		return first
	}
	return second
}

// Min32 returns the smallest 32-bit integer
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

// IsTty returns true is stdin is a terminal
func IsTty() bool {
	return int(C.isatty(C.int(os.Stdin.Fd()))) != 0
}

// TrimLen returns the length of trimmed rune array
func TrimLen(runes []rune) int {
	var i int
	for i = len(runes) - 1; i >= 0; i-- {
		char := runes[i]
		if char != ' ' && char != '\t' {
			break
		}
	}
	// Completely empty
	if i < 0 {
		return 0
	}

	var j int
	for j = 0; j < len(runes); j++ {
		char := runes[j]
		if char != ' ' && char != '\t' {
			break
		}
	}
	return i - j + 1
}

// ExecCommand executes the given command with $SHELL
func ExecCommand(command string) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	return exec.Command(shell, "-c", command)
}
