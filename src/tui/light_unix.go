//go:build !windows

package tui

/*
#include <sys/ioctl.h>
#include <unistd.h>
#include <termios.h>

// Simple wrapper to call the ioctl
int native_get_winsize(int fd, struct winsize *ws) {
    return ioctl(fd, TIOCGWINSZ, ws);
}

// Wrapper to call the stable POSIX tcgetattr function
int get_terminal_state(int fd, struct termios *t) {
    return tcgetattr(fd, t);
}

// Helper to manually set the terminal to raw mode using Haiku's C headers
int haiku_make_raw(int fd, struct termios *old) {
    struct termios raw;
    if (tcgetattr(fd, old) != 0) return -1;
    raw = *old;
    
    // POSIX raw mode flags
    raw.c_iflag &= ~(IGNBRK | BRKINT | PARMRK | ISTRIP | INLCR | IGNCR | ICRNL | IXON);
    raw.c_oflag &= ~OPOST;
    raw.c_lflag &= ~(ECHO | ECHONL | ICANON | ISIG | IEXTEN);
    raw.c_cflag &= ~(CSIZE | PARENB);
    raw.c_cflag |= CS8;
    raw.c_cc[VMIN] = 1;
    raw.c_cc[VTIME] = 0;

    return tcsetattr(fd, TCSAFLUSH, &raw);
}

int haiku_restore(int fd, struct termios *old) {
    return tcsetattr(fd, TCSAFLUSH, old);
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
	state	
}

type state struct {
	termios C.struct_termios
}

func MakeRaw(fd int) (*State, error) {
	var oldState state
	if C.haiku_make_raw(C.int(fd), &oldState.termios) != 0 {
		return nil, fmt.Errorf("failed to set raw mode on Haiku via CGO")
	}
	return &State{state: oldState}, nil
}

func Restore(fd int, oldState *State) error {
	//term_state := (*term.State)(unsafe.Pointer(oldState))
	if C.haiku_restore(C.int(fd), &oldState.termios) != 0 {
		return fmt.Errorf("haiku: failed to restore terminal")
	}
	return nil
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

	return &State{state{termios: t}}, nil
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
//	if err != nil {
//		return err
//	}
	if false {
		return err
	}
	r.origState = (*term.State)(unsafe.Pointer(origState))

	MakeRaw(fd)
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
	MakeRaw(r.fd())
}

func (r *LightRenderer) restoreTerminal() {

	state := (*State)(unsafe.Pointer(r.origState))
	Restore(r.fd(), state)
	//Restore(r.fd(), r.origState)
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
