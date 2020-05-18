#!/usr/bin/env python3

import os
import select
import termios
import struct
import time

STDIN = 0
STDOUT = 1

def main():
    fd = os.open('/dev/tty', os.O_RDONLY)

    size = os.get_terminal_size(fd)
    os.write(STDOUT, struct.pack('<LL', size.columns, size.lines))

    old = termios.tcgetattr(fd)
    new = list(old)
    new[0] = new[0] & ~(termios.ICRNL)
    new[3] = new[3] & ~(termios.ECHO | termios.ICANON | termios.ISIG)

    try:
        termios.tcsetattr(fd, termios.TCSAFLUSH, new)

        while True:
            r, _, _ = select.select([STDIN, fd], [], [])
            if fd in r:
                try:
                    buf = os.read(fd, 64)
                except:
                    break
                if len(buf) == 0:
                    break
                os.write(STDOUT, buf)
            if STDIN in r:
                os.write(STDOUT, b'\x00')
                break
    finally:
        termios.tcsetattr(fd, termios.TCSAFLUSH, old)

if __name__ == '__main__':
    main()
