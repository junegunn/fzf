package rawterm

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"syscall"
	"unsafe"
)

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if e != 0 {
		return e
	}
	return nil
}

func getTermios(fd int, t *syscall.Termios) error {
	return ioctl(uintptr(fd), uintptr(TERMGET), uintptr(unsafe.Pointer(t)))
}

func setTermios(fd int, t *syscall.Termios) error {
	return ioctl(uintptr(fd), uintptr(TERMSET), uintptr(unsafe.Pointer(t)))
}

func writeTermSizeRequest(fd int) error {
	req := []byte("\033[6n")
	n, err := syscall.Write(fd, req)
	if err != nil {
		return err
	}
	if n != 4 {
		return errors.New("write truncated")
	}
	return nil
}

func readTermSizeResponse(fd int) (row, col int, err error) {
	var (
		n   int
		res [128]byte
		buf []byte
	)
	n, err = syscall.Read(fd, res[:])
	if err != nil {
		return
	}
	if n <= 0 {
		err = errors.New("failed to read from terminal")
		return
	}

	escape := []byte("\033[")
	semicolon := []byte(";")
	R := []byte("R")

	buf = append(buf, res[:n]...)
	if !bytes.HasPrefix(buf, escape) || !bytes.HasSuffix(buf, R) || !bytes.Contains(buf, semicolon) {
		err = errors.New("unexpected terminal response")
		return
	}

	buf = bytes.TrimPrefix(buf, escape)
	buf = bytes.TrimSuffix(buf, R)
	components := bytes.Split(buf, semicolon)
	row, err = strconv.Atoi(string(components[0]))
	col, err = strconv.Atoi(string(components[1]))
	return
}

func GetCurRowCol(fd int) (row, col int, err error) {
	var saved syscall.Termios

	err = getTermios(fd, &saved)
	if err != nil {
		return
	}

	temp := saved
	temp.Lflag &^= syscall.ICANON | syscall.ECHO
	temp.Cflag &^= syscall.CREAD

	err = setTermios(fd, &temp)
	if err != nil {
		return
	}

	err = writeTermSizeRequest(fd)
	if err != nil {
		return
	}

	row, col, err = readTermSizeResponse(fd)
	if err != nil {
		return
	}

	err = setTermios(fd, &saved)
	return
}

func main() {
	row, col, err := GetCurRowCol(0)
	if err != nil {
		fmt.Println("error: ", err)
	} else {
		fmt.Println("(", row, ",", col, ")")
	}
}
