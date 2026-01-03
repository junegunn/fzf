//go:build !windows

package tui

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/junegunn/fzf/src/util"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func IsLightRendererSupported() bool {
	return true
}

func (r *LightRenderer) DefaultTheme() *ColorTheme {
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

func (r *LightRenderer) initPlatform() (err error) {
	r.origState, err = term.MakeRaw(r.fd())
	return err
}

func (r *LightRenderer) closePlatform() {
	r.ttyout.Close()
}

func openTty(ttyDefault string, mode int) (*os.File, error) {
	var in *os.File
	var err error
	if len(ttyDefault) > 0 {
		in, err = os.OpenFile(ttyDefault, mode, 0)
	}
	if in == nil || err != nil || ttyDefault != DefaultTtyDevice && !util.IsTty(in) {
		tty := ttyname()
		if len(tty) > 0 {
			if in, err := os.OpenFile(tty, mode, 0); err == nil {
				return in, nil
			}
		}
		if ttyDefault != DefaultTtyDevice {
			if in, err = os.OpenFile(DefaultTtyDevice, mode, 0); err == nil {
				return in, nil
			}
		}
		return nil, errors.New("failed to open " + DefaultTtyDevice)
	}
	return in, nil
}

func openTtyIn(ttyDefault string) (*os.File, error) {
	return openTty(ttyDefault, syscall.O_RDONLY)
}

func openTtyOut(ttyDefault string) (*os.File, error) {
	return openTty(ttyDefault, syscall.O_WRONLY)
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
	var err error
	bytes := []byte{}
	for tries := range offsetPollTries {
		bytes, _, err = r.getBytesInternal(false, bytes, tries > 0)
		if err != nil {
			return -1, -1
		}

		offsets := offsetRegexp.FindSubmatch(bytes)
		if len(offsets) > 3 {
			// Add anything we skipped over to the input buffer
			r.buffer = append(r.buffer, offsets[1]...)
			return atoi(string(offsets[2]), 0) - 1, atoi(string(offsets[3]), 0) - 1
		}
	}
	return -1, -1
}

func (r *LightRenderer) getch(cancellable bool, nonblock bool) (int, getCharResult) {
	b := make([]byte, 1)
	fd := r.fd()
	getter := func() (int, getCharResult) {
		_, err := util.Read(fd, b)
		if err != nil {
			return 0, getCharError
		}
		return int(b[0]), getCharSuccess
	}
	if nonblock || !cancellable {
		util.SetNonblock(r.ttyin, nonblock)
		return getter()
	}

	rpipe, wpipe, err := os.Pipe()
	if err != nil {
		// Fallback to blocking read without cancellation
		return getter()
	}
	r.mutex.Lock()
	r.cancel = func() {
		wpipe.Write([]byte{0})
	}
	r.mutex.Unlock()
	defer func() {
		r.mutex.Lock()
		r.cancel = nil
		rpipe.Close()
		wpipe.Close()
		r.mutex.Unlock()
	}()

	for {
		var rfds unix.FdSet
		cancelFd := int(rpipe.Fd())
		rfds.Bits[fd/64] |= 1 << (fd % 64)
		rfds.Bits[cancelFd/64] |= 1 << (cancelFd % 64)
		maxFd := max(fd, cancelFd)

		_, err := unix.Select(maxFd+1, &rfds, nil, nil, nil)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return 0, getCharError
		}

		// Cancel pipe triggered
		if rfds.Bits[cancelFd/64]&(1<<(cancelFd%64)) != 0 {
			return 0, getCharCancelled
		}

		// Data available
		if rfds.Bits[fd/64]&(1<<(fd%64)) != 0 {
			return getter()
		}
	}
}

func (r *LightRenderer) Size() TermSize {
	ws, err := unix.IoctlGetWinsize(int(r.ttyin.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return TermSize{}
	}
	return TermSize{int(ws.Row), int(ws.Col), int(ws.Xpixel), int(ws.Ypixel)}
}
