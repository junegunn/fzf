package util

import (
	"math"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
)

var _runeWidths = make(map[rune]int)

// RuneWidth returns rune width
func RuneWidth(r rune, prefixWidth int, tabstop int) int {
	if r == '\t' {
		return tabstop - prefixWidth%tabstop
	} else if w, found := _runeWidths[r]; found {
		return w
	} else if r == '\n' || r == '\r' {
		return 1
	}
	w := runewidth.RuneWidth(r)
	_runeWidths[r] = w
	return w
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

// IsTty returns true is stdin is a terminal
func IsTty() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}
