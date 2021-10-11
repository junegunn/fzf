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
	pusher      func([]byte) bool
	eventBox    *util.EventBox
	delimNil    bool
	event       int32
	finChan     chan bool
	mutex       sync.Mutex
	exec        *exec.Cmd
	command     *string
	killed      bool
	wait        bool
	dereference bool
}

// NewReader returns new Reader object
func NewReader(pusher func([]byte) bool, eventBox *util.EventBox, delimNil bool, wait bool, dereference bool) *Reader {
	return &Reader{pusher, eventBox, delimNil, int32(EvtReady), make(chan bool, 1), sync.Mutex{}, nil, nil, false, wait, dereference}
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
	w := r.newWalker()
	return w.Walk(".", fn, cb) == nil
}

// the walker library does not provide types, so those we use are repeated here
type walkerFunction func(path string, mode os.FileInfo) error
type walkerErrorCallback func(pathname string, err error) error

// Walker scans filesystem tree and reports files and dirs to walker function fn.
type walker interface {
	Walk(path string, fn walkerFunction, cb walkerErrorCallback) error
}

// Returns proper walker implementation based on current config.
func (r *Reader) newWalker() walker {
	if r.dereference {
		return newSymlinkWalker()
	} else {
		return &saracenWalker{}
	}
}

/*
	saracenWalker is the original implementation
*/
type saracenWalker struct{}

func (w *saracenWalker) Walk(path string, fn walkerFunction, cb walkerErrorCallback) error {
	opt := walkerLib.WithErrorCallback(cb)
	return walkerLib.Walk(path, fn, opt)
}

/*
	symlinkWalker is same as saracen, but follows symlinks
	- safe for concurrent use
	- works like man in the middle between saracen walker and fzf's implementation of walker function fn
*/
type symlinkWalker struct {
	mutex            sync.Mutex
	seenDirs         map[string]bool
	downstreamWalker saracenWalker
}

func newSymlinkWalker() *symlinkWalker {
	return &symlinkWalker{
		sync.Mutex{},
		make(map[string]bool),
		saracenWalker{},
	}
}

/*
	arg path: absolute or relative path of directory to walk
		- recursive calls can change this parameter to whatever path it needs
	arg fn:
		- callback function that will receive found files, prefixed with (original) path arg
		- on recursive calls this function will be chain of functions
			- e.g. for two symlink jumps: renameFn -> renameFn -> fn
			- this function chain is then hooked and sent to saracen: hookFn -> renameFn -> renameFn -> fn
		- caveat: the function fn receives path and os.FileInfo of the files and
			dirs, but since the path argument is translated by renameFn and
			os.FileInfo is taken from symlink's target absolute path (by saracen),
			these two may not report same filename and mode of root dirs (but the arguments
			always refer to the same filesystem node)
				- for current implementation this means that symlink is reported
					to fzf with its own symlink path and the os.FileInfo will
					report directory mode (or possibly file mode), but not symbolic link mode
				- this is useful and intentional, but easy to overlook
	arg cb: error callback (note saracen is not filtering errors encountered
		until reporting root dir through this function)

	Cycles are prevented by making sure this walker won't step in one directory twice.
*/
func (w *symlinkWalker) Walk(path string, fn walkerFunction, cb walkerErrorCallback) error {
	var hookFn walkerFunction

	basePath := path
	hookFn = func(path string, fi os.FileInfo) error {
		//TODO can't use io/fs.ModeSymlink constant, because go <1.16 will fail
		// with "src/reader.go:7:2: package io/fs is not in GOROOT (/Users/runner/
		// hostedtoolcache/go/1.14.15/x64/src/io/fs)"
		const ModeSymlink uint32 = 1 << 27

		mode := fi.Mode()
		isSymlink := uint32(mode)&ModeSymlink != 0
		isDir := mode.IsDir()

		// take a note of visited directories
		if isDir {
			canonicalPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			w.mutex.Lock()
			if w.seenDirs[canonicalPath] {
				w.mutex.Unlock()
				return filepath.SkipDir
			} else {
				w.seenDirs[canonicalPath] = true
				w.mutex.Unlock()
			}
		}

		// jump to symlink target
		if isSymlink {
			// we can't assume path is relative (or absolute)
			relPath, err := filepath.Rel(basePath, path)
			if err != nil {
				return err
			}

			// find the real path
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			realPath, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				return err
			}

			/*
				renameFn + fn chain:
				- responsible to translate symlink's target absolute path to
					whatever fzf was requesting
				- basePath: absolute path of the root node of the tree containing the symlink
				- relPath: path to the symlink file relative to the root node
				- rp: path to the subfile or subdir in symlink's target tree, relative to its own root

				- note that os.FileInfo is passed as is
			*/
			fn := func(p string, fi os.FileInfo) error {
				rp, err := filepath.Rel(realPath, p)
				if err != nil {
					return err
				}
				return fn(filepath.Join(basePath, relPath, rp), fi)
			}

			// we need to walk the symlink target explicitly
			return w.Walk(realPath, fn, cb)
		}

		// call downstream
		return fn(path, fi)
	}

	// call downstream
	return w.downstreamWalker.Walk(path, hookFn, cb)
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
