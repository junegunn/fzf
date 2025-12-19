package util

import (
	"math"
	"os"
	"strconv"
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

// Constrain32 limits the given 32-bit integer with the upper and lower bounds
func Constrain32(val int32, minimum int32, maximum int32) int32 {
	return max(min(val, maximum), minimum)
}

// Constrain limits the given integer with the upper and lower bounds
func Constrain(val int, minimum int, maximum int) int {
	return max(min(val, maximum), minimum)
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

// IsTty returns true if the file is a terminal
func IsTty(file *os.File) bool {
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// RunOnce runs the given function only once
func RunOnce(f func()) func() {
	once := Once(true)
	return func() {
		if once() {
			f()
		}
	}
}

// Once returns a function that returns the specified boolean value only once
func Once(nextResponse bool) func() bool {
	state := nextResponse
	return func() bool {
		prevState := state
		state = !nextResponse
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

// ToKebabCase converts the given CamelCase string to kebab-case
func ToKebabCase(s string) string {
	name := ""
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			name += "-"
		}
		name += string(r)
	}
	return strings.ToLower(name)
}

// CompareVersions compares two version strings
func CompareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	atoi := func(s string) int {
		n, e := strconv.Atoi(s)
		if e != nil {
			return 0
		}
		return n
	}

	for i := 0; i < max(len(parts1), len(parts2)); i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1 = atoi(parts1[i])
		}
		if i < len(parts2) {
			p2 = atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}
	return 0
}
