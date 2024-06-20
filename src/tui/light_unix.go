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

func (r *LightRenderer) initPlatform() (err error) {
	r.origState, err = term.MakeRaw(r.fd())
	return err
}

func (r *LightRenderer) closePlatform() {
	r.ttyout.Close()
}

func openTty(mode int) (*os.File, error) {
	in, err := os.OpenFile(consoleDevice, mode, 0)
	if err != nil {
		tty := ttyname()
		if len(tty) > 0 {
			if in, err := os.OpenFile(tty, mode, 0); err == nil {
				return in, nil
			}
		}
		return nil, errors.New("failed to open " + consoleDevice)
	}
	return in, nil
}

func openTtyIn() (*os.File, error) {
	return openTty(syscall.O_RDONLY)
}

func openTtyOut() (*os.File, error) {
	return openTty(syscall.O_WRONLY)
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
	for tries := 0; tries < offsetPollTries; tries++ {
		bytes, err = r.getBytesInternal(bytes, tries > 0)
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
	ws, err := unix.IoctlGetWinsize(int(r.ttyin.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return TermSize{}
	}
	return TermSize{int(ws.Row), int(ws.Col), int(ws.Xpixel), int(ws.Ypixel)}
}
