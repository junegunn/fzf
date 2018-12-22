// +build !openbsd

package protector

// Protect calls OS specific protections like pledge on OpenBSD
func Protect() {
	return
}
