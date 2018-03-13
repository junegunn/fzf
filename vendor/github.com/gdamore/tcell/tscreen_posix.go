// +build solaris

// Copyright 2015 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"os"
	"os/signal"
	"syscall"
)

// #include <termios.h>
// #include <sys/ioctl.h>
//
// int getwinsize(int fd, int *cols, int *rows) {
// #if defined TIOCGWINSZ
//	struct winsize w;
//	if (ioctl(fd, TIOCGWINSZ, &w) < 0) {
//		return (-1);
//	}
//	*cols = w.ws_col;
//	*rows = w.ws_row;
//	return (0);
// #else
//	return (-1);
// #endif
// }
//
// int getbaud(struct termios *tios) {
//     switch (cfgetospeed(tios)) {
// #ifdef B0
//     case B0: return (0);
// #endif
// #ifdef B50
//     case B50: return (50);
// #endif
// #ifdef B75
//     case B75: return (75);
// #endif
// #ifdef B110
//	case B110: return (110);
// #endif
// #ifdef B134
//	case B134: return (134);
// #endif
// #ifdef B150
//	case B150: return (150);
// #endif
// #ifdef B200
//	case B200: return (200);
// #endif
// #ifdef B300
//	case B300: return (300);
// #endif
// #ifdef B600
//	case B600: return (600);
// #endif
// #ifdef B1200
//	case B1200: return (1200);
// #endif
// #ifdef B1800
//	case B1800: return (1800);
// #endif
// #ifdef B2400
//	case B2400: return (2400);
// #endif
// #ifdef B4800
//	case B4800: return (4800);
// #endif
// #ifdef B9600
//	case B9600: return (9600);
// #endif
// #ifdef B19200
//	case B19200: return (19200);
// #endif
// #ifdef B38400
//	case B38400: return (38400);
// #endif
// #ifdef B57600
//	case B57600: return (57600);
// #endif
// #ifdef B76800
//	case B76800: return (76800);
// #endif
// #ifdef B115200
//	case B115200: return (115200);
// #endif
// #ifdef B153600
//	case B153600: return (153600);
// #endif
// #ifdef B230400
//	case B230400: return (230400);
// #endif
// #ifdef B307200
//	case B307200: return (307200);
// #endif
// #ifdef B460800
//	case B460800: return (460800);
// #endif
// #ifdef B921600
//	case B921600: return (921600);
// #endif
//	}
//	return (0);
// }
import "C"

type termiosPrivate struct {
	tios C.struct_termios
}

func (t *tScreen) termioInit() error {
	var e error
	var rv C.int
	var newtios C.struct_termios
	var fd C.int

	if t.in, e = os.OpenFile("/dev/tty", os.O_RDONLY, 0); e != nil {
		goto failed
	}
	if t.out, e = os.OpenFile("/dev/tty", os.O_WRONLY, 0); e != nil {
		goto failed
	}

	t.tiosp = &termiosPrivate{}

	fd = C.int(t.out.Fd())
	if rv, e = C.tcgetattr(fd, &t.tiosp.tios); rv != 0 {
		goto failed
	}
	t.baud = int(C.getbaud(&t.tiosp.tios))
	newtios = t.tiosp.tios
	newtios.c_iflag &^= C.IGNBRK | C.BRKINT | C.PARMRK |
		C.ISTRIP | C.INLCR | C.IGNCR |
		C.ICRNL | C.IXON
	newtios.c_oflag &^= C.OPOST
	newtios.c_lflag &^= C.ECHO | C.ECHONL | C.ICANON |
		C.ISIG | C.IEXTEN
	newtios.c_cflag &^= C.CSIZE | C.PARENB
	newtios.c_cflag |= C.CS8

	// We wake up at the earliest of 100 msec or when data is received.
	// We need to wake up frequently to permit us to exit cleanly and
	// close file descriptors on systems like Darwin, where close does
	// cause a wakeup.  (Probably we could reasonably increase this to
	// something like 1 sec or 500 msec.)
	newtios.c_cc[C.VMIN] = 0
	newtios.c_cc[C.VTIME] = 1

	if rv, e = C.tcsetattr(fd, C.TCSANOW|C.TCSAFLUSH, &newtios); rv != 0 {
		goto failed
	}

	signal.Notify(t.sigwinch, syscall.SIGWINCH)

	if w, h, e := t.getWinSize(); e == nil && w != 0 && h != 0 {
		t.cells.Resize(w, h)
	}

	return nil

failed:
	if t.in != nil {
		t.in.Close()
	}
	if t.out != nil {
		t.out.Close()
	}
	return e
}

func (t *tScreen) termioFini() {

	signal.Stop(t.sigwinch)

	<-t.indoneq

	if t.out != nil {
		fd := C.int(t.out.Fd())
		C.tcsetattr(fd, C.TCSANOW|C.TCSAFLUSH, &t.tiosp.tios)
		t.out.Close()
	}
	if t.in != nil {
		t.in.Close()
	}
}

func (t *tScreen) getWinSize() (int, int, error) {
	var cx, cy C.int
	if r, e := C.getwinsize(C.int(t.out.Fd()), &cx, &cy); r != 0 {
		return 0, 0, e
	}
	return int(cx), int(cy), nil
}
