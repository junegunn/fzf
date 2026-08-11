//go:build !windows

package tui

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

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

// waitReadable reports whether the tty has something to read before the
// deadline. A terminal that does not recognize a query answers nothing at all,
// so the read that follows must be able to stop waiting, or fzf would wait for
// the user to press a key instead of drawing itself. The timeout is generous
// because a terminal that does answer exceeds it only when the link is slow
// enough to be unusable anyway.
func (r *LightRenderer) waitReadable(timeout time.Duration) bool {
	fd := r.fd()
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false
		}
		var rfds unix.FdSet
		if fd >= len(rfds.Bits)*unix.NFDBITS {
			return false
		}
		rfds.Set(fd)
		// Recomputed each time round: Linux select rewrites the timeout with
		// the time left, other systems leave it alone
		tv := unix.NsecToTimeval(int64(remaining))
		n, err := unix.Select(fd+1, &rfds, nil, nil, &tv)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			// Nothing was confirmed readable, and the read that follows
			// reports the failure if the fd is really broken
			return false
		}
		return n > 0
	}
}

// queryTerminal sends every query in a single write and reads until the last
// one is answered. Returns the submatches of each reply, nil for a query the
// terminal ignored. Replies are cut out of what we read as they are recognized,
// so whatever is left over is input the user typed during the round trip.
func (r *LightRenderer) queryTerminal(queries []termQuery) [][][]byte {
	for _, query := range queries {
		r.csi(query.seq)
	}
	r.flush()

	replies := make([][][]byte, len(queries))
	buffer := []byte{}
	for tries := range offsetPollTries {
		// Only the first read blocks, so that is the one to put a bound on
		if tries == 0 && !r.waitReadable(queryTimeout) {
			return replies
		}

		var err error
		buffer, _, err = r.getBytesInternal(false, buffer, tries > 0)
		if err != nil {
			return replies
		}

		for idx, query := range queries {
			if replies[idx] != nil {
				continue
			}
			loc := query.reply.FindSubmatchIndex(buffer)
			if loc == nil {
				continue
			}
			groups := make([][]byte, len(loc)/2)
			for group := range groups {
				if loc[group*2] >= 0 {
					groups[group] = buffer[loc[group*2]:loc[group*2+1]]
				}
			}
			replies[idx] = groups
			// Capping the prefix makes append copy, leaving groups valid
			buffer = append(buffer[:loc[0]:loc[0]], buffer[loc[1]:]...)
		}

		if replies[len(queries)-1] != nil {
			break
		}
	}

	r.buffer = append(r.buffer, buffer...)
	return replies
}

func (r *LightRenderer) queryStartup() (row int, col int, pasteWasSet *bool) {
	replies := r.queryTerminal(startupQueries)
	before, paste, after := replies[0], replies[1], replies[2]

	if paste != nil && paste[1][0] != '0' {
		// 1 = set, 3 = permanently set
		set := paste[1][0] == '1' || paste[1][0] == '3'
		pasteWasSet = &set
	}
	row, col = parseOffset(before)

	// The cursor moved right likely because terminal doesn't support DECRQM
	if row2, col2 := parseOffset(after); row >= 0 && row2 == row && col2 > col {
		r.flushRaw(strings.Repeat("\b \b", col2-col))
	}
	return
}

func parseOffset(reply [][]byte) (row int, col int) {
	if reply == nil {
		return -1, -1
	}
	return atoi(string(reply[1]), 0) - 1, atoi(string(reply[2]), 0) - 1
}

func (r *LightRenderer) findOffset() (row int, col int) {
	return parseOffset(r.queryTerminal([]termQuery{offsetQuery})[0])
}

func (r *LightRenderer) getch(cancellable bool, nonblock bool) (int, getCharResult) {
	fd := r.fd()
	getter := func() (int, getCharResult) {
		b := make([]byte, 1)
		util.SetNonblock(r.ttyin, nonblock)
		_, err := util.Read(fd, b)
		if err != nil {
			return 0, getCharError
		}
		return int(b[0]), getCharSuccess
	}
	if nonblock || !cancellable {
		return getter()
	}

	rpipe, wpipe, err := os.Pipe()
	if err != nil {
		// Fallback to blocking read without cancellation
		return getter()
	}
	r.setCancel(func() {
		wpipe.Write([]byte{0})
	})
	defer func() {
		r.setCancel(nil)
		rpipe.Close()
		wpipe.Close()
	}()

	cancelFd := int(rpipe.Fd())
	for range maxSelectTries {
		var rfds unix.FdSet
		limit := len(rfds.Bits) * unix.NFDBITS
		if fd >= limit || cancelFd >= limit {
			return getter()
		}

		rfds.Set(fd)
		rfds.Set(cancelFd)
		_, err := unix.Select(max(fd, cancelFd)+1, &rfds, nil, nil, nil)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return 0, getCharError
		}

		if rfds.IsSet(cancelFd) {
			return 0, getCharCancelled
		}

		if rfds.IsSet(fd) {
			return getter()
		}
	}
	return 0, getCharError
}

func (r *LightRenderer) Size() TermSize {
	ws, err := unix.IoctlGetWinsize(int(r.ttyin.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return TermSize{}
	}
	return TermSize{int(ws.Row), int(ws.Col), int(ws.Xpixel), int(ws.Ypixel)}
}
