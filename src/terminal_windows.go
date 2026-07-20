//go:build windows

package fzf

import (
	"context"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const resizePollInterval = 100 * time.Millisecond

type resizeSignal struct{}

func (resizeSignal) String() string { return "resize" }
func (resizeSignal) Signal()        {}

// Windows has no SIGWINCH, so poll the console screen buffer for window
// size changes instead.
func notifyOnResize(ctx context.Context, resizeChan chan<- os.Signal) {
	consoleOut, err := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err != nil {
		return
	}
	var info windows.ConsoleScreenBufferInfo
	if windows.GetConsoleScreenBufferInfo(windows.Handle(consoleOut), &info) != nil {
		syscall.Close(consoleOut)
		return
	}
	last := info.Window
	go func() {
		defer syscall.Close(consoleOut)
		ticker := time.NewTicker(resizePollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			if windows.GetConsoleScreenBufferInfo(windows.Handle(consoleOut), &info) != nil {
				continue
			}
			current := info.Window
			if current.Right-current.Left != last.Right-last.Left ||
				current.Bottom-current.Top != last.Bottom-last.Top {
				last = current
				select {
				case resizeChan <- resizeSignal{}:
				default:
				}
			}
		}
	}()
}

func notifyStop(p *os.Process) {
	// NOOP
}
