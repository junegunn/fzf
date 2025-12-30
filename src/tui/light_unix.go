//go:build !windows

package tui

/*
#include <sys/ioctl.h>
#include <unistd.h>

// Simple wrapper to call the ioctl
int native_get_winsize(int fd, struct winsize *ws) {
    return ioctl(fd, TIOCGWINSZ, ws);
}

// Wrapper to call the stable POSIX tcgetattr function
int get_terminal_state(int fd, struct termios *t) {
    return tcgetattr(fd, t);
}
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"errors"
	"unsafe"

	"github.com/junegunn/fzf/src/util"
//	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// State wraps the C termios structure to keep it compatible with your existing code.
type State struct {
	termios C.struct_termios
}

// Attempt to replace:
// Ref: https://cs.opensource.google/go/x/term/+/refs/tags/v0.38.0:term_unix.go
// func getState(fd int) (*State, error) {
//	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
//	if err != nil {
//		return nil, err
//	}
//
//	return &State{state{termios: *termios}}, nil
//}

func GetState(fd int) (*State, error) {
	var t C.struct_termios
	
	// Call the C wrapper. 0 is success, -1 is failure.
	if C.get_terminal_state(C.int(fd), &t) != 0 {
		return nil, errors.New("failed to get terminal state (tcgetattr)")
	}

	return &State{termios: t}, nil
}

// Winsize matches the POSIX terminal window size structure
type Winsize struct {
	Rows   uint16
	Cols   uint16
	Xpixel uint16
	Ypixel uint16
}

func IsLightRendererSupported() bool {
	return true
}

func (r *LightRenderer) defaultTheme() *ColorTheme {
	if strings.Contains(os.Getenv("TERM"), "256") {
		return Dark256
	}
	colors, err := exec.Command("tput", "colors").Output()
	if err == nil && atoi(strings.TrimSpace(string(colors)), 16) > 16 {
		return Dark256
	}
	return Default16
}

func (r *LightRenderer) fd() int {
	return int(r.ttyin.Fd())
}

func (r *LightRenderer) initPlatform() error {
	fd := r.fd()
	origState, err := GetState(fd)
	if err != nil {
		return err
	}
	r.origState = (*term.State)(unsafe.Pointer(origState))

	term.MakeRaw(fd)
	return nil
}

func (r *LightRenderer) closePlatform() {
	// NOOP
}

func openTtyIn() *os.File {
	in, err := os.OpenFile(consoleDevice, syscall.O_RDONLY, 0)
	if err != nil {
		tty := ttyname()
		if len(tty) > 0 {
			if in, err := os.OpenFile(tty, syscall.O_RDONLY, 0); err == nil {
				return in
			}
		}
		fmt.Fprintln(os.Stderr, "Failed to open "+consoleDevice)
		os.Exit(2)
	}
	return in
}

func (r *LightRenderer) setupTerminal() {
	term.MakeRaw(r.fd())
}

func (r *LightRenderer) restoreTerminal() {
	term.Restore(r.fd(), r.origState)
}

func (r *LightRenderer) updateTerminalSize() {
	width, height, err := term.GetSize(r.fd())

	if err == nil {
		r.width = width
		r.height = r.maxHeightFunc(height)
	} else {
		r.width = getEnv("COLUMNS", defaultWidth)
		r.height = r.maxHeightFunc(getEnv("LINES", defaultHeight))
	}
}

func (r *LightRenderer) findOffset() (row int, col int) {
	r.csi("6n")
	r.flush()
	bytes := []byte{}
	for tries := 0; tries < offsetPollTries; tries++ {
		bytes = r.getBytesInternal(bytes, tries > 0)
		offsets := offsetRegexp.FindSubmatch(bytes)
		if len(offsets) > 3 {
			// Add anything we skipped over to the input buffer
			r.buffer = append(r.buffer, offsets[1]...)
			return atoi(string(offsets[2]), 0) - 1, atoi(string(offsets[3]), 0) - 1
		}
	}
	return -1, -1
}

func (r *LightRenderer) getch(nonblock bool) (int, bool) {
	b := make([]byte, 1)
	fd := r.fd()
	util.SetNonblock(r.ttyin, nonblock)
	_, err := util.Read(fd, b)
	if err != nil {
		return 0, false
	}
	return int(b[0]), true
}

func (r *LightRenderer) Size() TermSize {
	var ws C.struct_winsize
	res := C.native_get_winsize(C.int(r.ttyin.Fd()), &ws)

	if res != 0 {
		return TermSize{}
	}

	return TermSize{int(ws.ws_row), int(ws.ws_col), int(ws.ws_xpixel), int(ws.ws_ypixel)}
	
//	var cRows, cCols C.int
//	rs := C.get_terminal_size(C.int(r.ttyin.Fd()), &cRows, &cCols)

//	ws, err := unix.IoctlGetWinsize(int(r.ttyin.Fd()), unix.TIOCGWINSZ)
//	if err != nil {
//		return TermSize{}
//	}
//	return TermSize{int(ws.Row), int(ws.Col), int(ws.Xpixel), int(ws.Ypixel)}
//	return TermSize{0,0,0,0}
}
