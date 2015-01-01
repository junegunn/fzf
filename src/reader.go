package fzf

// #include <unistd.h>
import "C"

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const DEFAULT_COMMAND = "find * -path '*/\\.*' -prune -o -type f -print -o -type l -print 2> /dev/null"

type Reader struct {
	pusher   func(string)
	eventBox *EventBox
}

func (r *Reader) ReadSource() {
	if int(C.isatty(C.int(os.Stdin.Fd()))) != 0 {
		cmd := os.Getenv("FZF_DEFAULT_COMMAND")
		if len(cmd) == 0 {
			cmd = DEFAULT_COMMAND
		}
		r.readFromCommand(cmd)
	} else {
		r.readFromStdin()
	}
	r.eventBox.Set(EVT_READ_FIN, nil)
}

func (r *Reader) feed(src io.Reader) {
	if scanner := bufio.NewScanner(src); scanner != nil {
		for scanner.Scan() {
			r.pusher(scanner.Text())
			r.eventBox.Set(EVT_READ_NEW, nil)
		}
	}
}

func (r *Reader) readFromStdin() {
	r.feed(os.Stdin)
}

func (r *Reader) readFromCommand(cmd string) {
	arg := fmt.Sprintf("%q", cmd)
	listCommand := exec.Command("sh", "-c", arg[1:len(arg)-1])
	out, err := listCommand.StdoutPipe()
	if err != nil {
		return
	}
	err = listCommand.Start()
	if err != nil {
		return
	}
	defer listCommand.Wait()
	r.feed(out)
}
