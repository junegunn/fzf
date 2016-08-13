package util

import (
	"unicode/utf8"
)

type Chars struct {
	runes []rune
	bytes []byte
}

// ToChars converts byte array into rune array
func ToChars(bytea []byte) Chars {
	var runes []rune
	ascii := true
	numBytes := len(bytea)
	for i := 0; i < numBytes; {
		if bytea[i] < utf8.RuneSelf {
			if !ascii {
				runes = append(runes, rune(bytea[i]))
			}
			i++
		} else {
			if ascii {
				ascii = false
				runes = make([]rune, i, numBytes)
				for j := 0; j < i; j++ {
					runes[j] = rune(bytea[j])
				}
			}
			r, sz := utf8.DecodeRune(bytea[i:])
			i += sz
			runes = append(runes, r)
		}
	}
	if ascii {
		return Chars{bytes: bytea}
	}
	return Chars{runes: runes}
}

func RunesToChars(runes []rune) Chars {
	return Chars{runes: runes}
}

func (chars *Chars) Get(i int) rune {
	if chars.runes != nil {
		return chars.runes[i]
	}
	return rune(chars.bytes[i])
}

func (chars *Chars) Length() int {
	if chars.runes != nil {
		return len(chars.runes)
	}
	return len(chars.bytes)
}

// TrimLength returns the length after trimming leading and trailing whitespaces
func (chars *Chars) TrimLength() int {
	var i int
	len := chars.Length()
	for i = len - 1; i >= 0; i-- {
		char := chars.Get(i)
		if char != ' ' && char != '\t' {
			break
		}
	}
	// Completely empty
	if i < 0 {
		return 0
	}

	var j int
	for j = 0; j < len; j++ {
		char := chars.Get(j)
		if char != ' ' && char != '\t' {
			break
		}
	}
	return i - j + 1
}

func (chars *Chars) TrailingWhitespaces() int {
	whitespaces := 0
	for i := chars.Length() - 1; i >= 0; i-- {
		char := chars.Get(i)
		if char != ' ' && char != '\t' {
			break
		}
		whitespaces++
	}
	return whitespaces
}

func (chars *Chars) ToString() string {
	if chars.runes != nil {
		return string(chars.runes)
	}
	return string(chars.bytes)
}

func (chars *Chars) ToRunes() []rune {
	if chars.runes != nil {
		return chars.runes
	}
	runes := make([]rune, len(chars.bytes))
	for idx, b := range chars.bytes {
		runes[idx] = rune(b)
	}
	return runes
}
