package fzf

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charlievieth/fastwalk"
	"github.com/junegunn/fzf/src/util"
)

// Reader reads from command or standard input
type Reader struct {
	pusher   func([]byte) bool
	executor *util.Executor
	eventBox *util.EventBox
	delimNil bool
	event    int32
	finChan  chan bool
	mutex    sync.Mutex
	killed   bool
	termFunc func()
	command  *string
	wait     bool
}

// NewReader returns new Reader object
func NewReader(pusher func([]byte) bool, eventBox *util.EventBox, executor *util.Executor, delimNil bool, wait bool) *Reader {
	return &Reader{
		pusher,
		executor,
		eventBox,
		delimNil,
		int32(EvtReady),
		make(chan bool, 1),
		sync.Mutex{},
		false,
		func() { os.Stdin.Close() },
		nil,
		wait}
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
	r.killed = true
	if r.termFunc != nil {
		r.termFunc()
		r.termFunc = nil
	}
	r.mutex.Unlock()
}

func (r *Reader) restart(command commandSpec, environ []string, readyChan chan bool) {
	r.event = int32(EvtReady)
	r.startEventPoller()
	success := r.readFromCommand(command.command, environ, func() {
		readyChan <- true
	})
	r.fin(success)
	removeFiles(command.tempFiles)
}

func (r *Reader) readChannel(inputChan chan string) bool {
	for {
		item, more := <-inputChan
		if !more {
			break
		}
		if r.pusher([]byte(item)) {
			atomic.StoreInt32(&r.event, int32(EvtReadNew))
		}
	}
	return true
}

// ReadSource reads data from the default command or from standard input
func (r *Reader) ReadSource(inputChan chan string, roots []string, opts walkerOpts, ignores []string, initCmd string, initEnv []string, readyChan chan bool) {
	r.startEventPoller()
	var success bool
	signalReady := func() {
		if readyChan != nil {
			readyChan <- true
		}
	}
	if inputChan != nil {
		signalReady()
		success = r.readChannel(inputChan)
	} else if len(initCmd) > 0 {
		success = r.readFromCommand(initCmd, initEnv, signalReady)
	} else if util.IsTty(os.Stdin) {
		cmd := os.Getenv("FZF_DEFAULT_COMMAND")
		if len(cmd) == 0 {
			signalReady()
			success = r.readFiles(roots, opts, ignores)
		} else {
			success = r.readFromCommand(cmd, initEnv, signalReady)
		}
	} else {
		signalReady()
		success = r.readFromStdin()
	}
	r.fin(success)
}

func (r *Reader) feed(src io.Reader) {
	/*
		readerSlabSize, ae := strconv.Atoi(os.Getenv("SLAB_KB"))
		if ae != nil {
			readerSlabSize = 128 * 1024
		} else {
			readerSlabSize *= 1024
		}
		readerBufferSize, be := strconv.Atoi(os.Getenv("BUF_KB"))
		if be != nil {
			readerBufferSize = 64 * 1024
		} else {
			readerBufferSize *= 1024
		}
	*/

	delim := byte('\n')
	trimCR := util.IsWindows()
	if r.delimNil {
		delim = '\000'
		trimCR = false
	}

	slab := make([]byte, readerSlabSize)
	leftover := []byte{}
	var err error
	for {
		n := 0
		scope := slab[:min(len(slab), readerBufferSize)]
		for range 100 {
			n, err = src.Read(scope)
			if n > 0 || err != nil {
				break
			}
		}

		// We're not making any progress after 100 tries. Stop.
		if n == 0 {
			break
		}

		buf := slab[:n]
		slab = slab[n:]

		for len(buf) > 0 {
			if i := bytes.IndexByte(buf, delim); i >= 0 {
				// Found the delimiter
				slice := buf[:i+1]
				buf = buf[i+1:]
				if trimCR && len(slice) >= 2 && slice[len(slice)-2] == byte('\r') {
					slice = slice[:len(slice)-2]
				} else {
					slice = slice[:len(slice)-1]
				}
				if len(leftover) > 0 {
					slice = append(leftover, slice...)
					leftover = []byte{}
				}
				if (err == nil || len(slice) > 0) && r.pusher(slice) {
					atomic.StoreInt32(&r.event, int32(EvtReadNew))
				}
			} else {
				// Could not find the delimiter in the buffer
				//   NOTE: We can further optimize this by keeping track of the cursor
				//   position in the slab so that a straddling item that doesn't go
				//   beyond the boundary of a slab doesn't need to be copied to
				//   another buffer. However, the performance gain is negligible in
				//   practice (< 0.1%) and is not
				//   worth the added complexity.
				leftover = append(leftover, buf...)
				break
			}
		}

		if err == io.EOF {
			leftover = append(leftover, buf...)
			break
		}

		if len(slab) == 0 {
			slab = make([]byte, readerSlabSize)
		}
	}
	if len(leftover) > 0 && r.pusher(leftover) {
		atomic.StoreInt32(&r.event, int32(EvtReadNew))
	}
}

func (r *Reader) readFromStdin() bool {
	r.feed(os.Stdin)
	return true
}

func isSymlinkToDir(path string, de os.DirEntry) bool {
	if de.Type()&fs.ModeSymlink == 0 {
		return false
	}
	if s, err := os.Stat(path); err == nil {
		return s.IsDir()
	}
	return false
}

func trimPath(path string) string {
	bytes := stringBytes(path)

	for len(bytes) > 1 && bytes[0] == '.' && (bytes[1] == '/' || bytes[1] == '\\') {
		bytes = bytes[2:]
	}

	if len(bytes) == 0 {
		return "."
	}

	return byteString(bytes)
}

func (r *Reader) readFiles(roots []string, opts walkerOpts, ignores []string) bool {
	conf := fastwalk.Config{
		Follow: opts.follow,
		// Use forward slashes when running a Windows binary under WSL or MSYS
		ToSlash: fastwalk.DefaultToSlash(),
		Sort:    fastwalk.SortFilesFirst,
	}
	ignoresBase := []string{}
	ignoresFull := []string{}
	ignoresSuffix := []string{}
	sep := string(os.PathSeparator)
	if _, ok := os.LookupEnv("MSYSTEM"); ok {
		sep = "/"
	}
	for _, ignore := range ignores {
		if strings.ContainsRune(ignore, os.PathSeparator) {
			if strings.HasPrefix(ignore, sep) {
				ignoresSuffix = append(ignoresSuffix, ignore)
			} else {
				// 'foo/bar' should match
				// * 'foo/bar'
				// * 'baz/foo/bar'
				// * but NOT 'bazfoo/bar'
				ignoresFull = append(ignoresFull, ignore)
				ignoresSuffix = append(ignoresSuffix, sep+ignore)
			}
		} else {
			ignoresBase = append(ignoresBase, ignore)
		}
	}
	fn := func(path string, de os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		path = trimPath(path)
		if path != "." {
			isDir := de.IsDir()
			if isDir || opts.follow && isSymlinkToDir(path, de) {
				base := filepath.Base(path)
				if !opts.hidden && base[0] == '.' && base != ".." {
					return filepath.SkipDir
				}
				if slices.Contains(ignoresBase, base) {
					return filepath.SkipDir
				}
				if slices.Contains(ignoresFull, path) {
					return filepath.SkipDir
				}
				for _, ignore := range ignoresSuffix {
					if strings.HasSuffix(path, ignore) {
						return filepath.SkipDir
					}
				}
				if path != sep {
					path += sep
				}
			}
			if ((opts.file && !isDir) || (opts.dir && isDir)) && r.pusher(stringBytes(path)) {
				atomic.StoreInt32(&r.event, int32(EvtReadNew))
			}
		}
		r.mutex.Lock()
		defer r.mutex.Unlock()
		if r.killed {
			return context.Canceled
		}
		return nil
	}
	noerr := true
	for _, root := range roots {
		noerr = noerr && (fastwalk.Walk(&conf, root, fn) == nil)
	}
	return noerr
}

func (r *Reader) readFromCommand(command string, environ []string, signalReady func()) bool {
	r.mutex.Lock()

	r.killed = false
	r.termFunc = nil
	r.command = &command
	exec := r.executor.ExecCommand(command, true)
	if environ != nil {
		exec.Env = environ
	}
	execOut, err := exec.StdoutPipe()
	if err != nil || exec.Start() != nil {
		signalReady()
		r.mutex.Unlock()
		return false
	}

	// Function to call to terminate the running command
	r.termFunc = func() {
		execOut.Close()
		util.KillCommand(exec)
	}

	signalReady()
	r.mutex.Unlock()

	r.feed(execOut)
	return exec.Wait() == nil
}
