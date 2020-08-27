//+build windows

package tui

import (
	"os"
	"syscall"

	"github.com/junegunn/fzf/src/util"
	"golang.org/x/sys/windows"
)

var (
	consoleFlagsInput  = uint32(windows.ENABLE_VIRTUAL_TERMINAL_INPUT | windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_EXTENDED_FLAGS)
	consoleFlagsOutput = uint32(windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.ENABLE_PROCESSED_OUTPUT | windows.DISABLE_NEWLINE_AUTO_RETURN)
)

// IsLightRendererSupported checks to see if the Light renderer is supported
func IsLightRendererSupported() bool {
	var oldState uint32
	// enable vt100 emulation (https://docs.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences)
	if windows.GetConsoleMode(windows.Stderr, &oldState) != nil {
		return false
	}
	// attempt to set mode to determine if we support VT 100 codes. This will work on newer Windows 10
	// version:
	canSetVt100 := windows.SetConsoleMode(windows.Stderr, oldState|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
	var checkState uint32
	if windows.GetConsoleMode(windows.Stderr, &checkState) != nil ||
		(checkState&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING {
		return false
	}
	windows.SetConsoleMode(windows.Stderr, oldState)
	return canSetVt100
}

func (r *LightRenderer) defaultTheme() *ColorTheme {
	// the getenv check is borrowed from here: https://github.com/gdamore/tcell/commit/0c473b86d82f68226a142e96cc5a34c5a29b3690#diff-b008fcd5e6934bf31bc3d33bf49f47d8R178:
	if !IsLightRendererSupported() || os.Getenv("ConEmuPID") != "" || os.Getenv("TCELL_TRUECOLOR") == "disable" {
		return Default16
	}
	return Dark256
}

func (r *LightRenderer) initPlatform() error {
	//outHandle := windows.Stdout
	outHandle, _ := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	// enable vt100 emulation (https://docs.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences)
	if err := windows.GetConsoleMode(windows.Handle(outHandle), &r.origStateOutput); err != nil {
		return err
	}
	r.outHandle = uintptr(outHandle)
	inHandle, _ := syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if err := windows.GetConsoleMode(windows.Handle(inHandle), &r.origStateInput); err != nil {
		return err
	}
	r.inHandle = uintptr(inHandle)

	r.setupTerminal()

	// channel for non-blocking reads. Buffer to make sure
	// we get the ESC sets:
	r.ttyinChannel = make(chan byte, 12)

	// the following allows for non-blocking IO.
	// syscall.SetNonblock() is a NOOP under Windows.
	go func() {
		fd := int(r.inHandle)
		b := make([]byte, 1)
		for {
			// HACK: if run from PSReadline, something resets ConsoleMode to remove ENABLE_VIRTUAL_TERMINAL_INPUT.
			_ = windows.SetConsoleMode(windows.Handle(r.inHandle), consoleFlagsInput)

			_, err := util.Read(fd, b)
			if err == nil {
				r.ttyinChannel <- b[0]
			}
		}
	}()

	return nil
}

func (r *LightRenderer) closePlatform() {
	windows.SetConsoleMode(windows.Handle(r.outHandle), r.origStateOutput)
	windows.SetConsoleMode(windows.Handle(r.inHandle), r.origStateInput)
}

func openTtyIn() *os.File {
	// not used
	return nil
}

func (r *LightRenderer) setupTerminal() error {
	if err := windows.SetConsoleMode(windows.Handle(r.outHandle), consoleFlagsOutput); err != nil {
		return err
	}
	return windows.SetConsoleMode(windows.Handle(r.inHandle), consoleFlagsInput)
}

func (r *LightRenderer) restoreTerminal() error {
	if err := windows.SetConsoleMode(windows.Handle(r.inHandle), r.origStateInput); err != nil {
		return err
	}
	return windows.SetConsoleMode(windows.Handle(r.outHandle), r.origStateOutput)
}

func (r *LightRenderer) updateTerminalSize() {
	var bufferInfo windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(r.outHandle), &bufferInfo); err != nil {
		r.width = getEnv("COLUMNS", defaultWidth)
		r.height = r.maxHeightFunc(getEnv("LINES", defaultHeight))

	} else {
		r.width = int(bufferInfo.Window.Right - bufferInfo.Window.Left)
		r.height = r.maxHeightFunc(int(bufferInfo.Window.Bottom - bufferInfo.Window.Top))
	}
}

func (r *LightRenderer) findOffset() (row int, col int) {
	var bufferInfo windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(r.outHandle), &bufferInfo); err != nil {
		return -1, -1
	}
	return int(bufferInfo.CursorPosition.X), int(bufferInfo.CursorPosition.Y)
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
