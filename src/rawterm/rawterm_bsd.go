// +build darwin dragonfly freebsd netbsd openbsd

package rawterm

import "syscall"

const (
    TERMSET = syscall.TIOCSETA
    TERMGET = syscall.TIOCGETA
)
