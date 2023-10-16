//go:build openbsd

package protector

import "golang.org/x/sys/unix"

// Protect calls OS specific protections like pledge on OpenBSD
func Protect() {
	unix.PledgePromises("stdio rpath tty proc exec inet")
}
