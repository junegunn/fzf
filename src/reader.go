package fzf

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/junegunn/fzf/src/util"
	walkerLib "github.com/saracen/walker"
)

// Reader reads from command or standard input
type Reader struct {
	pusher   func([]byte) bool
	eventBox *util.EventBox
	delimNil bool
	event    int32
	finChan  chan bool
	mutex    sync.Mutex
	exec     *exec.Cmd
	command  *string
	killed   bool
	wait     bool
}

// NewReader returns new Reader object
func NewReader(pusher func([]byte) bool, eventBox *util.EventBox, delimNil bool, wait bool) *Reader {
	return &Reader{pusher, eventBox, delimNil, int32(EvtReady), make(chan bool, 1), sync.Mutex{}, nil, nil, false, wait}
}

func (r *Reader) startEventPoller() {
	go func() {
		ptr := &r.event
		pollInterval := readerPollIntervalMin
		for {
			if atomic.CompareAndSwapInt32(ptr, int32(EvtReadNew), int32(EvtReady)) {
				r.eventBox.Set(EvtReadNew, (*string)(nil))
				pollInterval = readerPollIntervalMin
			} else if atomic.LoadInt32(ptr) == int32(EvtReadFin) {
				if r.wait {
					r.finChan <- true
				}
				return
			} else {
				pollInterval += readerPollIntervalStep
				if pollInterval > readerPollIntervalMax {
					pollInterval = readerPollIntervalMax
				}
			}
			time.Sleep(pollInterval)
		}
	}()
}

func (r *Reader) fin(success bool) {
	atomic.StoreInt32(&r.event, int32(EvtReadFin))
	if r.wait {
		<-r.finChan
	}

	r.mutex.Lock()
	ret := r.command
	if success || r.killed {
		ret = nil
	}
	r.mutex.Unlock()

	r.eventBox.Set(EvtReadFin, ret)
}

func (r *Reader) terminate() {
	r.mutex.Lock()
	defer func() { r.mutex.Unlock() }()

	r.killed = true
	if r.exec != nil && r.exec.Process != nil {
		util.KillCommand(r.exec)
	} else if defaultCommand != "" {
		os.Stdin.Close()
	}
}

func (r *Reader) restart(command string) {
	r.event = int32(EvtReady)
	r.startEventPoller()
	success := r.readFromCommand(nil, command)
	r.fin(success)
}

// ReadSource reads data from the default command or from standard input
func (r *Reader) ReadSource() {
	r.startEventPoller()
	var success bool
	if util.IsTty() {
		// The default command for *nix requires bash
		shell := "bash"
		cmd := os.Getenv("FZF_DEFAULT_COMMAND")
		if len(cmd) == 0 {
			if defaultCommand != "" {
				success = r.readFromCommand(&shell, defaultCommand)
			} else {
				success = r.readFiles()
			}
		} else {
			success = r.readFromCommand(nil, cmd)
		}
	} else {
		success = r.readFromStdin()
	}
	r.fin(success)
}

func (r *Reader) feed(src io.Reader) {
	delim := byte('\n')
	if r.delimNil {
		delim = '\000'
	}
	reader := bufio.NewReaderSize(src, readerBufferSize)
	for {
		// ReadBytes returns err != nil if and only if the returned data does not
		// end in delim.
		bytea, err := reader.ReadBytes(delim)
		byteaLen := len(bytea)
		if byteaLen > 0 {
			if err == nil {
				// get rid of carriage return if under Windows:
				if util.IsWindows() && byteaLen >= 2 && bytea[byteaLen-2] == byte('\r') {
					bytea = bytea[:byteaLen-2]
				} else {
					bytea = bytea[:byteaLen-1]
				}
			}
			if r.pusher(bytea) {
				atomic.StoreInt32(&r.event, int32(EvtReadNew))
			}
		}
		if err != nil {
			break
		}
	}
}

func (r *Reader) readFromStdin() bool {
	r.feed(os.Stdin)
	return true
}

func (r *Reader) readFiles() bool {
	var fn walkerFunction
	var cb walkerErrorCallback
	r.killed = false

	/*
		A function called for each root, file and subdir; recursively.

		- arg path: relative path, expects:
			- "." for root
			- "./subdirs/subfiles" for subtree objects
		- arg mode: path's file info
	*/
	fn = func(path string, mode os.FileInfo) error {
		// simplify the path, e.g. drop the leading "./" if any
		path = filepath.Clean(path)

		//non root
		if path != "." {
			isDir := mode.Mode().IsDir()
			// skip hidden (subtree) dirs
			if isDir && filepath.Base(path)[0] == '.' {
				return filepath.SkipDir
			}
			// push files to fzf
			if !isDir && r.pusher([]byte(path)) {
				atomic.StoreInt32(&r.event, int32(EvtReadNew))
			}
		}
		// continue (and test for interrupt)
		r.mutex.Lock()
		defer r.mutex.Unlock()
		if r.killed {
			return context.Canceled
		}
		return nil
	}
	/*
		The error chain is below, note:
		- errors up to fzf.fn(".") call are not filtered via cb()
		- subsequent errors (most notably errors for subfiles, fzf.fn("./..."))
			are filtered, possibly multiple times
		- gowalk() spins out goroutines

		fzf.readFiles()
			<- walkerLib.Walk()
				<- walkerLib.WalkWithContext()
					<- fzf.fn(".")
					<- go walkerLib.gowalk()
						<- fzf.cb() <- walkerLib.readdir()
										<- os.calls
										<- walkerLib.walk()
											<- fzf.fn("./...")
											<- fzf.cb() <- walkerLib.readdir()
	*/
	cb = func(pathname string, err error) error {
		// ignore the error
		return nil
	}

	// walk the working dir
	w := newWalker()
	return w.Walk(".", fn, cb) == nil
}

// the walker library does not provide types, so those we use are repeated here
type walkerFunction func(path string, mode os.FileInfo) error
type walkerErrorCallback func(pathname string, err error) error
type walker interface {
	Walk(path string, fn walkerFunction, cb walkerErrorCallback) error
}

func newWalker() walker {
	return &saracenWalker{}
}

// saracenWalker is the original implementation
type saracenWalker struct{}

func (w *saracenWalker) Walk(path string, fn walkerFunction, cb walkerErrorCallback) error {
	opt := walkerLib.WithErrorCallback(cb)
	return walkerLib.Walk(path, fn, opt)
}

func (r *Reader) readFromCommand(shell *string, command string) bool {
	r.mutex.Lock()
	r.killed = false
	r.command = &command
	if shell != nil {
		r.exec = util.ExecCommandWith(*shell, command, true)
	} else {
		r.exec = util.ExecCommand(command, true)
	}
	out, err := r.exec.StdoutPipe()
	if err != nil {
		r.mutex.Unlock()
		return false
	}
	err = r.exec.Start()
	r.mutex.Unlock()
	if err != nil {
		return false
	}
	r.feed(out)
	return r.exec.Wait() == nil
}
