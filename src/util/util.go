package util

import (
	"math"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rivo/uniseg"
)

// StringWidth returns string width where each CR/LF character takes 1 column
func StringWidth(s string) int {
	return uniseg.StringWidth(s) + strings.Count(s, "\n") + strings.Count(s, "\r")
}

// RunesWidth returns runes width
func RunesWidth(runes []rune, prefixWidth int, tabstop int, limit int) (int, int) {
	width := 0
	gr := uniseg.NewGraphemes(string(runes))
	idx := 0
	for gr.Next() {
		rs := gr.Runes()
		var w int
		if len(rs) == 1 && rs[0] == '\t' {
			w = tabstop - (prefixWidth+width)%tabstop
		} else {
			w = StringWidth(string(rs))
		}
		width += w
		if width > limit {
			return width, idx
		}
		idx += len(rs)
	}
	return width, -1
}

// Truncate returns the truncated runes and its width
func Truncate(input string, limit int) ([]rune, int) {
	runes := []rune{}
	width := 0
	gr := uniseg.NewGraphemes(input)
	for gr.Next() {
		rs := gr.Runes()
		w := StringWidth(string(rs))
		if width+w > limit {
			return runes, width
		}
		width += w
		runes = append(runes, rs...)
	}
	return runes, width
}

// Max returns the largest integer
func Max(first int, second int) int {
	if first >= second {
		return first
	}
	return second
}

// Max16 returns the largest integer
func Max16(first int16, second int16) int16 {
	if first >= second {
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

func AsUint16(val int) uint16 {
	if val > math.MaxUint16 {
		return math.MaxUint16
	} else if val < 0 {
		return 0
	}
	return uint16(val)
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

// IsTty returns true if stdin is a terminal
func IsTty() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

// ToTty returns true if stdout is a terminal
func ToTty() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// Once returns a function that returns the specified boolean value only once
func Once(nextResponse bool) func() bool {
	state := nextResponse
	return func() bool {
		prevState := state
		state = false
		return prevState
	}
}

// RepeatToFill repeats the given string to fill the given width
func RepeatToFill(str string, length int, limit int) string {
	times := limit / length
	rest := limit % length
	output := strings.Repeat(str, times)
	if rest > 0 {
		for _, r := range str {
			rest -= uniseg.StringWidth(string(r))
			if rest < 0 {
				break
			}
			output += string(r)
			if rest == 0 {
				break
			}
		}
	}
	return output
}
