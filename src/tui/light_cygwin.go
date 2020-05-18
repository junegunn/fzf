//+build windows
//+build cygwin

package tui

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
)

func IsLightRendererSupported() bool {
	return true
}

func (r *LightRenderer) initPlatform() error {
	var procAttr os.ProcAttr
	path, err := exec.LookPath("sh.exe")
	if err != nil {
		return err
	}
	cygwin_tty_out_r, cygwin_tty_out_w, _ := os.Pipe()
	cygwin_tty_in_r, cygwin_tty_in_w, _ := os.Pipe()
	procAttr.Dir, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	procAttr.Files = []*os.File{cygwin_tty_in_r, cygwin_tty_out_w, os.Stderr}
	os.StartProcess(path, []string{"sh.exe", "-c", "./cygwin_tty.py"}, &procAttr)

	// channel for non-blocking reads. Buffer to make sure
	// we get the ESC sets:
	r.ttyinChannel = make(chan byte, 12)
	r.cygwin_tty_out_r = cygwin_tty_in_w

	b := make([]byte, 4)
	cygwin_tty_out_r.Read(b)
	r.width = int(binary.LittleEndian.Uint32(b))
	cygwin_tty_out_r.Read(b)
	r.height = r.maxHeightFunc(int(binary.LittleEndian.Uint32(b)))

	// the following allows for non-blocking IO.
	// syscall.SetNonblock() is a NOOP under Windows.
	go func() {
		b := make([]byte, 1)
		for {
			_, err := cygwin_tty_out_r.Read(b)
			if err == nil {
				r.ttyinChannel <- b[0]
			}
		}
	}()

	return nil
}

func (r *LightRenderer) closePlatform() {
	for len(r.ttyinChannel) > 0 {
	  <-r.ttyinChannel
	}
	r.cygwin_tty_out_r.Close()
	<-r.ttyinChannel
}

func openTtyIn() *os.File {
	// not used
	return nil
}

func (r *LightRenderer) setupTerminal() {
}

func (r *LightRenderer) restoreTerminal() {
}

func (r *LightRenderer) updateTerminalSize() {
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
	if nonblock {
		select {
		case bc := <-r.ttyinChannel:
			return int(bc), true
		default:
			return 0, false
		}
	} else {
		bc := <-r.ttyinChannel
		return int(bc), true
	}
}
