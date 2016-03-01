package util

// #include <unistd.h>
import "C"

import (
	"os"
	"os/exec"
	"time"
	"unicode/utf8"
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

// TrimRight returns rune array with trailing white spaces cut off
func TrimRight(runes []rune) []rune {
	var i int
	for i = len(runes) - 1; i >= 0; i-- {
		char := runes[i]
		if char != ' ' && char != '\t' {
			break
		}
	}
	return runes[0 : i+1]
}

// BytesToRunes converts byte array into rune array
func BytesToRunes(bytea []byte) []rune {
	runes := make([]rune, 0, len(bytea))
	for i := 0; i < len(bytea); {
		if bytea[i] < utf8.RuneSelf {
			runes = append(runes, rune(bytea[i]))
			i++
		} else {
			r, sz := utf8.DecodeRune(bytea[i:])
			i += sz
			runes = append(runes, r)
		}
	}
	return runes
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
