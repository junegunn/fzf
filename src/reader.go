package fzf

import (
	"bufio"
	"io"
	"os"
	"os/exec"

	"github.com/junegunn/fzf/src/util"
)

const defaultCommand = `find * -path '*/\.*' -prune -o -type f -print -o -type l -print 2> /dev/null`

// Reader reads from command or standard input
type Reader struct {
	pusher   func(string)
	eventBox *util.EventBox
}

// ReadSource reads data from the default command or from standard input
func (r *Reader) ReadSource() {
	if util.IsTty() {
		cmd := os.Getenv("FZF_DEFAULT_COMMAND")
		if len(cmd) == 0 {
			cmd = defaultCommand
		}
		r.readFromCommand(cmd)
	} else {
		r.readFromStdin()
	}
	r.eventBox.Set(EvtReadFin, nil)
}

func (r *Reader) feed(src io.Reader) {
	if scanner := bufio.NewScanner(src); scanner != nil {
		for scanner.Scan() {
			r.pusher(scanner.Text())
			r.eventBox.Set(EvtReadNew, nil)
		}
	}
}

func (r *Reader) readFromStdin() {
	r.feed(os.Stdin)
}

func (r *Reader) readFromCommand(cmd string) {
	listCommand := exec.Command("sh", "-c", cmd)
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
