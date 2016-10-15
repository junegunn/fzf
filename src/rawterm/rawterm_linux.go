// +build linux
package rawterm

import "syscall"

const (
    TERMSET = syscall.TCSETS
    TERMGET = syscall.TCGETS
)
