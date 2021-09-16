package util

import (
	"runtime"
)

type osName string

const OS = osName(runtime.GOOS)

// Returns one of the arguments, based on operating system
func (os osName) Sieve(onUnix interface{}, onWindows interface{}) interface{} {
	if os == "windows" {
		return onWindows
	} else {
		return onUnix
	}
}

