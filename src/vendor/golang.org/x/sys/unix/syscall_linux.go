// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Linux system calls.
// This file is compiled as ordinary Go code,
// but it is also input to mksyscall,
// which parses the //sys lines and generates system call stubs.
// Note that sometimes we use a lowercase //sys name and
// wrap it in our own nicer implementation.

package unix

import (
	"syscall"
	"unsafe"
)

/*
 * Wrapped
 */

func Access(path string, mode uint32) (err error) {
	return Faccessat(AT_FDCWD, path, mode, 0)
}

func Chmod(path string, mode uint32) (err error) {
	return Fchmodat(AT_FDCWD, path, mode, 0)
}

func Chown(path string, uid int, gid int) (err error) {
	return Fchownat(AT_FDCWD, path, uid, gid, 0)
}

func Creat(path string, mode uint32) (fd int, err error) {
	return Open(path, O_CREAT|O_WRONLY|O_TRUNC, mode)
}

//sys	Linkat(olddirfd int, oldpath string, newdirfd int, newpath string, flags int) (err error)

func Link(oldpath string, newpath string) (err error) {
	return Linkat(AT_FDCWD, oldpath, AT_FDCWD, newpath, 0)
}

func Mkdir(path string, mode uint32) (err error) {
	return Mkdirat(AT_FDCWD, path, mode)
}

func Mknod(path string, mode uint32, dev int) (err error) {
	return Mknodat(AT_FDCWD, path, mode, dev)
}

func Open(path string, mode int, perm uint32) (fd int, err error) {
	return openat(AT_FDCWD, path, mode|O_LARGEFILE, perm)
}

//sys	openat(dirfd int, path string, flags int, mode uint32) (fd int, err error)

func Openat(dirfd int, path string, flags int, mode uint32) (fd int, err error) {
	return openat(dirfd, path, flags|O_LARGEFILE, mode)
}

//sys	ppoll(fds *PollFd, nfds int, timeout *Timespec, sigmask *Sigset_t) (n int, err error)

func Ppoll(fds []PollFd, timeout *Timespec, sigmask *Sigset_t) (n int, err error) {
	if len(fds) == 0 {
		return ppoll(nil, 0, timeout, sigmask)
	}
	return ppoll(&fds[0], len(fds), timeout, sigmask)
}

//sys	Readlinkat(dirfd int, path string, buf []byte) (n int, err error)

func Readlink(path string, buf []byte) (n int, err error) {
	return Readlinkat(AT_FDCWD, path, buf)
}

func Rename(oldpath string, newpath string) (err error) {
	return Renameat(AT_FDCWD, oldpath, AT_FDCWD, newpath)
}

func Rmdir(path string) error {
	return Unlinkat(AT_FDCWD, path, AT_REMOVEDIR)
}

//sys	Symlinkat(oldpath string, newdirfd int, newpath string) (err error)

func Symlink(oldpath string, newpath string) (err error) {
	return Symlinkat(oldpath, AT_FDCWD, newpath)
}

func Unlink(path string) error {
	return Unlinkat(AT_FDCWD, path, 0)
}

//sys	Unlinkat(dirfd int, path string, flags int) (err error)

//sys	utimes(path string, times *[2]Timeval) (err error)

func Utimes(path string, tv []Timeval) error {
	if tv == nil {
		err := utimensat(AT_FDCWD, path, nil, 0)
		if err != ENOSYS {
			return err
		}
		return utimes(path, nil)
	}
	if len(tv) != 2 {
		return EINVAL
	}
	var ts [2]Timespec
	ts[0] = NsecToTimespec(TimevalToNsec(tv[0]))
	ts[1] = NsecToTimespec(TimevalToNsec(tv[1]))
	err := utimensat(AT_FDCWD, path, (*[2]Timespec)(unsafe.Pointer(&ts[0])), 0)
	if err != ENOSYS {
		return err
	}
	return utimes(path, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

//sys	utimensat(dirfd int, path string, times *[2]Timespec, flags int) (err error)

func UtimesNano(path string, ts []Timespec) error {
	if ts == nil {
		err := utimensat(AT_FDCWD, path, nil, 0)
		if err != ENOSYS {
			return err
		}
		return utimes(path, nil)
	}
	if len(ts) != 2 {
		return EINVAL
	}
	err := utimensat(AT_FDCWD, path, (*[2]Timespec)(unsafe.Pointer(&ts[0])), 0)
	if err != ENOSYS {
		return err
	}
	// If the utimensat syscall isn't available (utimensat was added to Linux
	// in 2.6.22, Released, 8 July 2007) then fall back to utimes
	var tv [2]Timeval
	for i := 0; i < 2; i++ {
		tv[i] = NsecToTimeval(TimespecToNsec(ts[i]))
	}
	return utimes(path, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

func UtimesNanoAt(dirfd int, path string, ts []Timespec, flags int) error {
	if ts == nil {
		return utimensat(dirfd, path, nil, flags)
	}
	if len(ts) != 2 {
		return EINVAL
	}
	return utimensat(dirfd, path, (*[2]Timespec)(unsafe.Pointer(&ts[0])), flags)
}

//sys	futimesat(dirfd int, path *byte, times *[2]Timeval) (err error)

func Futimesat(dirfd int, path string, tv []Timeval) error {
	pathp, err := BytePtrFromString(path)
	if err != nil {
		return err
	}
	if tv == nil {
		return futimesat(dirfd, pathp, nil)
	}
	if len(tv) != 2 {
		return EINVAL
	}
	return futimesat(dirfd, pathp, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

func Futimes(fd int, tv []Timeval) (err error) {
	// Believe it or not, this is the best we can do on Linux
	// (and is what glibc does).
	return Utimes("/proc/self/fd/"+itoa(fd), tv)
}

const ImplementsGetwd = true

//sys	Getcwd(buf []byte) (n int, err error)

func Getwd() (wd string, err error) {
	var buf [PathMax]byte
	n, err := Getcwd(buf[0:])
	if err != nil {
		return "", err
	}
	// Getcwd returns the number of bytes written to buf, including the NUL.
	if n < 1 || n > len(buf) || buf[n-1] != 0 {
		return "", EINVAL
	}
	return string(buf[0 : n-1]), nil
}

func Getgroups() (gids []int, err error) {
	n, err := getgroups(0, nil)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	// Sanity check group count.  Max is 1<<16 on Linux.
	if n < 0 || n > 1<<20 {
		return nil, EINVAL
	}

	a := make([]_Gid_t, n)
	n, err = getgroups(n, &a[0])
	if err != nil {
		return nil, err
	}
	gids = make([]int, n)
	for i, v := range a[0:n] {
		gids[i] = int(v)
	}
	return
}

func Setgroups(gids []int) (err error) {
	if len(gids) == 0 {
		return setgroups(0, nil)
	}

	a := make([]_Gid_t, len(gids))
	for i, v := range gids {
		a[i] = _Gid_t(v)
	}
	return setgroups(len(a), &a[0])
}

type WaitStatus uint32

// Wait status is 7 bits at bottom, either 0 (exited),
// 0x7F (stopped), or a signal number that caused an exit.
// The 0x80 bit is whether there was a core dump.
// An extra number (exit code, signal causing a stop)
// is in the high bits.  At least that's the idea.
// There are various irregularities.  For example, the
// "continued" status is 0xFFFF, distinguishing itself
// from stopped via the core dump bit.

const (
	mask    = 0x7F
	core    = 0x80
	exited  = 0x00
	stopped = 0x7F
	shift   = 8
)

func (w WaitStatus) Exited() bool { return w&mask == exited }

func (w WaitStatus) Signaled() bool { return w&mask != stopped && w&mask != exited }

func (w WaitStatus) Stopped() bool { return w&0xFF == stopped }

func (w WaitStatus) Continued() bool { return w == 0xFFFF }

func (w WaitStatus) CoreDump() bool { return w.Signaled() && w&core != 0 }

func (w WaitStatus) ExitStatus() int {
	if !w.Exited() {
		return -1
	}
	return int(w>>shift) & 0xFF
}

func (w WaitStatus) Signal() syscall.Signal {
	if !w.Signaled() {
		return -1
	}
	return syscall.Signal(w & mask)
}

func (w WaitStatus) StopSignal() syscall.Signal {
	if !w.Stopped() {
		return -1
	}
	return syscall.Signal(w>>shift) & 0xFF
}

func (w WaitStatus) TrapCause() int {
	if w.StopSignal() != SIGTRAP {
		return -1
	}
	return int(w>>shift) >> 8
}

//sys	wait4(pid int, wstatus *_C_int, options int, rusage *Rusage) (wpid int, err error)

func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error) {
	var status _C_int
	wpid, err = wait4(pid, &status, options, rusage)
	if wstatus != nil {
		*wstatus = WaitStatus(status)
	}
	return
}

func Mkfifo(path string, mode uint32) (err error) {
	return Mknod(path, mode|S_IFIFO, 0)
}

func (sa *SockaddrInet4) sockaddr() (unsafe.Pointer, _Socklen, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, 0, EINVAL
	}
	sa.raw.Family = AF_INET
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return unsafe.Pointer(&sa.raw), SizeofSockaddrInet4, nil
}

func (sa *SockaddrInet6) sockaddr() (unsafe.Pointer, _Socklen, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, 0, EINVAL
	}
	sa.raw.Family = AF_INET6
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	sa.raw.Scope_id = sa.ZoneId
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return unsafe.Pointer(&sa.raw), SizeofSockaddrInet6, nil
}

func (sa *SockaddrUnix) sockaddr() (unsafe.Pointer, _Socklen, error) {
	name := sa.Name
	n := len(name)
	if n >= len(sa.raw.Path) {
		return nil, 0, EINVAL
	}
	sa.raw.Family = AF_UNIX
	for i := 0; i < n; i++ {
		sa.raw.Path[i] = int8(name[i])
	}
	// length is family (uint16), name, NUL.
	sl := _Socklen(2)
	if n > 0 {
		sl += _Socklen(n) + 1
	}
	if sa.raw.Path[0] == '@' {
		sa.raw.Path[0] = 0
		// Don't count trailing NUL for abstract address.
		sl--
	}

	return unsafe.Pointer(&sa.raw), sl, nil
}

type SockaddrLinklayer struct {
	Protocol uint16
	Ifindex  int
	Hatype   uint16
	Pkttype  uint8
	Halen    uint8
	Addr     [8]byte
	raw      RawSockaddrLinklayer
}

func (sa *SockaddrLinklayer) sockaddr() (unsafe.Pointer, _Socklen, error) {
	if sa.Ifindex < 0 || sa.Ifindex > 0x7fffffff {
		return nil, 0, EINVAL
	}
	sa.raw.Family = AF_PACKET
	sa.raw.Protocol = sa.Protocol
	sa.raw.Ifindex = int32(sa.Ifindex)
	sa.raw.Hatype = sa.Hatype
	sa.raw.Pkttype = sa.Pkttype
	sa.raw.Halen = sa.Halen
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return unsafe.Pointer(&sa.raw), SizeofSockaddrLinklayer, nil
}

type SockaddrNetlink struct {
	Family uint16
	Pad    uint16
	Pid    uint32
	Groups uint32
	raw    RawSockaddrNetlink
}

func (sa *SockaddrNetlink) sockaddr() (unsafe.Pointer, _Socklen, error) {
	sa.raw.Family = AF_NETLINK
	sa.raw.Pad = sa.Pad
	sa.raw.Pid = sa.Pid
	sa.raw.Groups = sa.Groups
	return unsafe.Pointer(&sa.raw), SizeofSockaddrNetlink, nil
}

type SockaddrHCI struct {
	Dev     uint16
	Channel uint16
	raw     RawSockaddrHCI
}

func (sa *SockaddrHCI) sockaddr() (unsafe.Pointer, _Socklen, error) {
	sa.raw.Family = AF_BLUETOOTH
	sa.raw.Dev = sa.Dev
	sa.raw.Channel = sa.Channel
	return unsafe.Pointer(&sa.raw), SizeofSockaddrHCI, nil
}

// SockaddrCAN implements the Sockaddr interface for AF_CAN type sockets.
// The RxID and TxID fields are used for transport protocol addressing in
// (CAN_TP16, CAN_TP20, CAN_MCNET, and CAN_ISOTP), they can be left with
// zero values for CAN_RAW and CAN_BCM sockets as they have no meaning.
//
// The SockaddrCAN struct must be bound to the socket file descriptor
// using Bind before the CAN socket can be used.
//
//      // Read one raw CAN frame
//      fd, _ := Socket(AF_CAN, SOCK_RAW, CAN_RAW)
//      addr := &SockaddrCAN{Ifindex: index}
//      Bind(fd, addr)
//      frame := make([]byte, 16)
//      Read(fd, frame)
//
// The full SocketCAN documentation can be found in the linux kernel
// archives at: https://www.kernel.org/doc/Documentation/networking/can.txt
type SockaddrCAN struct {
	Ifindex int
	RxID    uint32
	TxID    uint32
	raw     RawSockaddrCAN
}

func (sa *SockaddrCAN) sockaddr() (unsafe.Pointer, _Socklen, error) {
	if sa.Ifindex < 0 || sa.Ifindex > 0x7fffffff {
		return nil, 0, EINVAL
	}
	sa.raw.Family = AF_CAN
	sa.raw.Ifindex = int32(sa.Ifindex)
	rx := (*[4]byte)(unsafe.Pointer(&sa.RxID))
	for i := 0; i < 4; i++ {
		sa.raw.Addr[i] = rx[i]
	}
	tx := (*[4]byte)(unsafe.Pointer(&sa.TxID))
	for i := 0; i < 4; i++ {
		sa.raw.Addr[i+4] = tx[i]
	}
	return unsafe.Pointer(&sa.raw), SizeofSockaddrCAN, nil
}

// SockaddrALG implements the Sockaddr interface for AF_ALG type sockets.
// SockaddrALG enables userspace access to the Linux kernel's cryptography
// subsystem. The Type and Name fields specify which type of hash or cipher
// should be used with a given socket.
//
// To create a file descriptor that provides access to a hash or cipher, both
// Bind and Accept must be used. Once the setup process is complete, input
// data can be written to the socket, processed by the kernel, and then read
// back as hash output or ciphertext.
//
// Here is an example of using an AF_ALG socket with SHA1 hashing.
// The initial socket setup process is as follows:
//
//      // Open a socket to perform SHA1 hashing.
//      fd, _ := unix.Socket(unix.AF_ALG, unix.SOCK_SEQPACKET, 0)
//      addr := &unix.SockaddrALG{Type: "hash", Name: "sha1"}
//      unix.Bind(fd, addr)
//      // Note: unix.Accept does not work at this time; must invoke accept()
//      // manually using unix.Syscall.
//      hashfd, _, _ := unix.Syscall(unix.SYS_ACCEPT, uintptr(fd), 0, 0)
//
// Once a file descriptor has been returned from Accept, it may be used to
// perform SHA1 hashing. The descriptor is not safe for concurrent use, but
// may be re-used repeatedly with subsequent Write and Read operations.
//
// When hashing a small byte slice or string, a single Write and Read may
// be used:
//
//      // Assume hashfd is already configured using the setup process.
//      hash := os.NewFile(hashfd, "sha1")
//      // Hash an input string and read the results. Each Write discards
//      // previous hash state. Read always reads the current state.
//      b := make([]byte, 20)
//      for i := 0; i < 2; i++ {
//          io.WriteString(hash, "Hello, world.")
//          hash.Read(b)
//          fmt.Println(hex.EncodeToString(b))
//      }
//      // Output:
//      // 2ae01472317d1935a84797ec1983ae243fc6aa28
//      // 2ae01472317d1935a84797ec1983ae243fc6aa28
//
// For hashing larger byte slices, or byte streams such as those read from
// a file or socket, use Sendto with MSG_MORE to instruct the kernel to update
// the hash digest instead of creating a new one for a given chunk and finalizing it.
//
//      // Assume hashfd and addr are already configured using the setup process.
//      hash := os.NewFile(hashfd, "sha1")
//      // Hash the contents of a file.
//      f, _ := os.Open("/tmp/linux-4.10-rc7.tar.xz")
//      b := make([]byte, 4096)
//      for {
//          n, err := f.Read(b)
//          if err == io.EOF {
//              break
//          }
//          unix.Sendto(hashfd, b[:n], unix.MSG_MORE, addr)
//      }
//      hash.Read(b)
//      fmt.Println(hex.EncodeToString(b))
//      // Output: 85cdcad0c06eef66f805ecce353bec9accbeecc5
//
// For more information, see: http://www.chronox.de/crypto-API/crypto/userspace-if.html.
type SockaddrALG struct {
	Type    string
	Name    string
	Feature uint32
	Mask    uint32
	raw     RawSockaddrALG
}

func (sa *SockaddrALG) sockaddr() (unsafe.Pointer, _Socklen, error) {
	// Leave room for NUL byte terminator.
	if len(sa.Type) > 13 {
		return nil, 0, EINVAL
	}
	if len(sa.Name) > 63 {
		return nil, 0, EINVAL
	}

	sa.raw.Family = AF_ALG
	sa.raw.Feat = sa.Feature
	sa.raw.Mask = sa.Mask

	typ, err := ByteSliceFromString(sa.Type)
	if err != nil {
		return nil, 0, err
	}
	name, err := ByteSliceFromString(sa.Name)
	if err != nil {
		return nil, 0, err
	}

	copy(sa.raw.Type[:], typ)
	copy(sa.raw.Name[:], name)

	return unsafe.Pointer(&sa.raw), SizeofSockaddrALG, nil
}

// SockaddrVM implements the Sockaddr interface for AF_VSOCK type sockets.
// SockaddrVM provides access to Linux VM sockets: a mechanism that enables
// bidirectional communication between a hypervisor and its guest virtual
// machines.
type SockaddrVM struct {
	// CID and Port specify a context ID and port address for a VM socket.
	// Guests have a unique CID, and hosts may have a well-known CID of:
	//  - VMADDR_CID_HYPERVISOR: refers to the hypervisor process.
	//  - VMADDR_CID_HOST: refers to other processes on the host.
	CID  uint32
	Port uint32
	raw  RawSockaddrVM
}

func (sa *SockaddrVM) sockaddr() (unsafe.Pointer, _Socklen, error) {
	sa.raw.Family = AF_VSOCK
	sa.raw.Port = sa.Port
	sa.raw.Cid = sa.CID

	return unsafe.Pointer(&sa.raw), SizeofSockaddrVM, nil
}

func anyToSockaddr(rsa *RawSockaddrAny) (Sockaddr, error) {
	switch rsa.Addr.Family {
	case AF_NETLINK:
		pp := (*RawSockaddrNetlink)(unsafe.Pointer(rsa))
		sa := new(SockaddrNetlink)
		sa.Family = pp.Family
		sa.Pad = pp.Pad
		sa.Pid = pp.Pid
		sa.Groups = pp.Groups
		return sa, nil

	case AF_PACKET:
		pp := (*RawSockaddrLinklayer)(unsafe.Pointer(rsa))
		sa := new(SockaddrLinklayer)
		sa.Protocol = pp.Protocol
		sa.Ifindex = int(pp.Ifindex)
		sa.Hatype = pp.Hatype
		sa.Pkttype = pp.Pkttype
		sa.Halen = pp.Halen
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil

	case AF_UNIX:
		pp := (*RawSockaddrUnix)(unsafe.Pointer(rsa))
		sa := new(SockaddrUnix)
		if pp.Path[0] == 0 {
			// "Abstract" Unix domain socket.
			// Rewrite leading NUL as @ for textual display.
			// (This is the standard convention.)
			// Not friendly to overwrite in place,
			// but the callers below don't care.
			pp.Path[0] = '@'
		}

		// Assume path ends at NUL.
		// This is not technically the Linux semantics for
		// abstract Unix domain sockets--they are supposed
		// to be uninterpreted fixed-size binary blobs--but
		// everyone uses this convention.
		n := 0
		for n < len(pp.Path) && pp.Path[n] != 0 {
			n++
		}
		bytes := (*[10000]byte)(unsafe.Pointer(&pp.Path[0]))[0:n]
		sa.Name = string(bytes)
		return sa, nil

	case AF_INET:
		pp := (*RawSockaddrInet4)(unsafe.Pointer(rsa))
		sa := new(SockaddrInet4)
		p := (*[2]byte)(unsafe.Pointer(&pp.Port))
		sa.Port = int(p[0])<<8 + int(p[1])
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil

	case AF_INET6:
		pp := (*RawSockaddrInet6)(unsafe.Pointer(rsa))
		sa := new(SockaddrInet6)
		p := (*[2]byte)(unsafe.Pointer(&pp.Port))
		sa.Port = int(p[0])<<8 + int(p[1])
		sa.ZoneId = pp.Scope_id
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil

	case AF_VSOCK:
		pp := (*RawSockaddrVM)(unsafe.Pointer(rsa))
		sa := &SockaddrVM{
			CID:  pp.Cid,
			Port: pp.Port,
		}
		return sa, nil
	}
	return nil, EAFNOSUPPORT
}

func Accept(fd int) (nfd int, sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	nfd, err = accept(fd, &rsa, &len)
	if err != nil {
		return
	}
	sa, err = anyToSockaddr(&rsa)
	if err != nil {
		Close(nfd)
		nfd = 0
	}
	return
}

func Accept4(fd int, flags int) (nfd int, sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	nfd, err = accept4(fd, &rsa, &len, flags)
	if err != nil {
		return
	}
	if len > SizeofSockaddrAny {
		panic("RawSockaddrAny too small")
	}
	sa, err = anyToSockaddr(&rsa)
	if err != nil {
		Close(nfd)
		nfd = 0
	}
	return
}

func Getsockname(fd int) (sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	if err = getsockname(fd, &rsa, &len); err != nil {
		return
	}
	return anyToSockaddr(&rsa)
}

func GetsockoptInet4Addr(fd, level, opt int) (value [4]byte, err error) {
	vallen := _Socklen(4)
	err = getsockopt(fd, level, opt, unsafe.Pointer(&value[0]), &vallen)
	return value, err
}

func GetsockoptIPMreq(fd, level, opt int) (*IPMreq, error) {
	var value IPMreq
	vallen := _Socklen(SizeofIPMreq)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptIPMreqn(fd, level, opt int) (*IPMreqn, error) {
	var value IPMreqn
	vallen := _Socklen(SizeofIPMreqn)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptIPv6Mreq(fd, level, opt int) (*IPv6Mreq, error) {
	var value IPv6Mreq
	vallen := _Socklen(SizeofIPv6Mreq)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptIPv6MTUInfo(fd, level, opt int) (*IPv6MTUInfo, error) {
	var value IPv6MTUInfo
	vallen := _Socklen(SizeofIPv6MTUInfo)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptICMPv6Filter(fd, level, opt int) (*ICMPv6Filter, error) {
	var value ICMPv6Filter
	vallen := _Socklen(SizeofICMPv6Filter)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptUcred(fd, level, opt int) (*Ucred, error) {
	var value Ucred
	vallen := _Socklen(SizeofUcred)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptTCPInfo(fd, level, opt int) (*TCPInfo, error) {
	var value TCPInfo
	vallen := _Socklen(SizeofTCPInfo)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func SetsockoptIPMreqn(fd, level, opt int, mreq *IPMreqn) (err error) {
	return setsockopt(fd, level, opt, unsafe.Pointer(mreq), unsafe.Sizeof(*mreq))
}

func Recvmsg(fd int, p, oob []byte, flags int) (n, oobn int, recvflags int, from Sockaddr, err error) {
	var msg Msghdr
	var rsa RawSockaddrAny
	msg.Name = (*byte)(unsafe.Pointer(&rsa))
	msg.Namelen = uint32(SizeofSockaddrAny)
	var iov Iovec
	if len(p) > 0 {
		iov.Base = (*byte)(unsafe.Pointer(&p[0]))
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		// receive at least one normal byte
		if len(p) == 0 {
			iov.Base = &dummy
			iov.SetLen(1)
		}
		msg.Control = (*byte)(unsafe.Pointer(&oob[0]))
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	if n, err = recvmsg(fd, &msg, flags); err != nil {
		return
	}
	oobn = int(msg.Controllen)
	recvflags = int(msg.Flags)
	// source address is only specified if the socket is unconnected
	if rsa.Addr.Family != AF_UNSPEC {
		from, err = anyToSockaddr(&rsa)
	}
	return
}

func Sendmsg(fd int, p, oob []byte, to Sockaddr, flags int) (err error) {
	_, err = SendmsgN(fd, p, oob, to, flags)
	return
}

func SendmsgN(fd int, p, oob []byte, to Sockaddr, flags int) (n int, err error) {
	var ptr unsafe.Pointer
	var salen _Socklen
	if to != nil {
		var err error
		ptr, salen, err = to.sockaddr()
		if err != nil {
			return 0, err
		}
	}
	var msg Msghdr
	msg.Name = (*byte)(unsafe.Pointer(ptr))
	msg.Namelen = uint32(salen)
	var iov Iovec
	if len(p) > 0 {
		iov.Base = (*byte)(unsafe.Pointer(&p[0]))
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		// send at least one normal byte
		if len(p) == 0 {
			iov.Base = &dummy
			iov.SetLen(1)
		}
		msg.Control = (*byte)(unsafe.Pointer(&oob[0]))
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	if n, err = sendmsg(fd, &msg, flags); err != nil {
		return 0, err
	}
	if len(oob) > 0 && len(p) == 0 {
		n = 0
	}
	return n, nil
}

// BindToDevice binds the socket associated with fd to device.
func BindToDevice(fd int, device string) (err error) {
	return SetsockoptString(fd, SOL_SOCKET, SO_BINDTODEVICE, device)
}

//sys	ptrace(request int, pid int, addr uintptr, data uintptr) (err error)

func ptracePeek(req int, pid int, addr uintptr, out []byte) (count int, err error) {
	// The peek requests are machine-size oriented, so we wrap it
	// to retrieve arbitrary-length data.

	// The ptrace syscall differs from glibc's ptrace.
	// Peeks returns the word in *data, not as the return value.

	var buf [sizeofPtr]byte

	// Leading edge.  PEEKTEXT/PEEKDATA don't require aligned
	// access (PEEKUSER warns that it might), but if we don't
	// align our reads, we might straddle an unmapped page
	// boundary and not get the bytes leading up to the page
	// boundary.
	n := 0
	if addr%sizeofPtr != 0 {
		err = ptrace(req, pid, addr-addr%sizeofPtr, uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return 0, err
		}
		n += copy(out, buf[addr%sizeofPtr:])
		out = out[n:]
	}

	// Remainder.
	for len(out) > 0 {
		// We use an internal buffer to guarantee alignment.
		// It's not documented if this is necessary, but we're paranoid.
		err = ptrace(req, pid, addr+uintptr(n), uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return n, err
		}
		copied := copy(out, buf[0:])
		n += copied
		out = out[copied:]
	}

	return n, nil
}

func PtracePeekText(pid int, addr uintptr, out []byte) (count int, err error) {
	return ptracePeek(PTRACE_PEEKTEXT, pid, addr, out)
}

func PtracePeekData(pid int, addr uintptr, out []byte) (count int, err error) {
	return ptracePeek(PTRACE_PEEKDATA, pid, addr, out)
}

func PtracePeekUser(pid int, addr uintptr, out []byte) (count int, err error) {
	return ptracePeek(PTRACE_PEEKUSR, pid, addr, out)
}

func ptracePoke(pokeReq int, peekReq int, pid int, addr uintptr, data []byte) (count int, err error) {
	// As for ptracePeek, we need to align our accesses to deal
	// with the possibility of straddling an invalid page.

	// Leading edge.
	n := 0
	if addr%sizeofPtr != 0 {
		var buf [sizeofPtr]byte
		err = ptrace(peekReq, pid, addr-addr%sizeofPtr, uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return 0, err
		}
		n += copy(buf[addr%sizeofPtr:], data)
		word := *((*uintptr)(unsafe.Pointer(&buf[0])))
		err = ptrace(pokeReq, pid, addr-addr%sizeofPtr, word)
		if err != nil {
			return 0, err
		}
		data = data[n:]
	}

	// Interior.
	for len(data) > sizeofPtr {
		word := *((*uintptr)(unsafe.Pointer(&data[0])))
		err = ptrace(pokeReq, pid, addr+uintptr(n), word)
		if err != nil {
			return n, err
		}
		n += sizeofPtr
		data = data[sizeofPtr:]
	}

	// Trailing edge.
	if len(data) > 0 {
		var buf [sizeofPtr]byte
		err = ptrace(peekReq, pid, addr+uintptr(n), uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return n, err
		}
		copy(buf[0:], data)
		word := *((*uintptr)(unsafe.Pointer(&buf[0])))
		err = ptrace(pokeReq, pid, addr+uintptr(n), word)
		if err != nil {
			return n, err
		}
		n += len(data)
	}

	return n, nil
}

func PtracePokeText(pid int, addr uintptr, data []byte) (count int, err error) {
	return ptracePoke(PTRACE_POKETEXT, PTRACE_PEEKTEXT, pid, addr, data)
}

func PtracePokeData(pid int, addr uintptr, data []byte) (count int, err error) {
	return ptracePoke(PTRACE_POKEDATA, PTRACE_PEEKDATA, pid, addr, data)
}

func PtraceGetRegs(pid int, regsout *PtraceRegs) (err error) {
	return ptrace(PTRACE_GETREGS, pid, 0, uintptr(unsafe.Pointer(regsout)))
}

func PtraceSetRegs(pid int, regs *PtraceRegs) (err error) {
	return ptrace(PTRACE_SETREGS, pid, 0, uintptr(unsafe.Pointer(regs)))
}

func PtraceSetOptions(pid int, options int) (err error) {
	return ptrace(PTRACE_SETOPTIONS, pid, 0, uintptr(options))
}

func PtraceGetEventMsg(pid int) (msg uint, err error) {
	var data _C_long
	err = ptrace(PTRACE_GETEVENTMSG, pid, 0, uintptr(unsafe.Pointer(&data)))
	msg = uint(data)
	return
}

func PtraceCont(pid int, signal int) (err error) {
	return ptrace(PTRACE_CONT, pid, 0, uintptr(signal))
}

func PtraceSyscall(pid int, signal int) (err error) {
	return ptrace(PTRACE_SYSCALL, pid, 0, uintptr(signal))
}

func PtraceSingleStep(pid int) (err error) { return ptrace(PTRACE_SINGLESTEP, pid, 0, 0) }

func PtraceAttach(pid int) (err error) { return ptrace(PTRACE_ATTACH, pid, 0, 0) }

func PtraceDetach(pid int) (err error) { return ptrace(PTRACE_DETACH, pid, 0, 0) }

//sys	reboot(magic1 uint, magic2 uint, cmd int, arg string) (err error)

func Reboot(cmd int) (err error) {
	return reboot(LINUX_REBOOT_MAGIC1, LINUX_REBOOT_MAGIC2, cmd, "")
}

func ReadDirent(fd int, buf []byte) (n int, err error) {
	return Getdents(fd, buf)
}

func direntIno(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(Dirent{}.Ino), unsafe.Sizeof(Dirent{}.Ino))
}

func direntReclen(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(Dirent{}.Reclen), unsafe.Sizeof(Dirent{}.Reclen))
}

func direntNamlen(buf []byte) (uint64, bool) {
	reclen, ok := direntReclen(buf)
	if !ok {
		return 0, false
	}
	return reclen - uint64(unsafe.Offsetof(Dirent{}.Name)), true
}

//sys	mount(source string, target string, fstype string, flags uintptr, data *byte) (err error)

func Mount(source string, target string, fstype string, flags uintptr, data string) (err error) {
	// Certain file systems get rather angry and EINVAL if you give
	// them an empty string of data, rather than NULL.
	if data == "" {
		return mount(source, target, fstype, flags, nil)
	}
	datap, err := BytePtrFromString(data)
	if err != nil {
		return err
	}
	return mount(source, target, fstype, flags, datap)
}

// Sendto
// Recvfrom
// Socketpair

/*
 * Direct access
 */
//sys	Acct(path string) (err error)
//sys	Adjtimex(buf *Timex) (state int, err error)
//sys	Chdir(path string) (err error)
//sys	Chroot(path string) (err error)
//sys	ClockGettime(clockid int32, time *Timespec) (err error)
//sys	Close(fd int) (err error)
//sys	Dup(oldfd int) (fd int, err error)
//sys	Dup3(oldfd int, newfd int, flags int) (err error)
//sysnb	EpollCreate(size int) (fd int, err error)
//sysnb	EpollCreate1(flag int) (fd int, err error)
//sysnb	EpollCtl(epfd int, op int, fd int, event *EpollEvent) (err error)
//sys	Exit(code int) = SYS_EXIT_GROUP
//sys	Faccessat(dirfd int, path string, mode uint32, flags int) (err error)
//sys	Fallocate(fd int, mode uint32, off int64, len int64) (err error)
//sys	Fchdir(fd int) (err error)
//sys	Fchmod(fd int, mode uint32) (err error)
//sys	Fchmodat(dirfd int, path string, mode uint32, flags int) (err error)
//sys	Fchownat(dirfd int, path string, uid int, gid int, flags int) (err error)
//sys	fcntl(fd int, cmd int, arg int) (val int, err error)
//sys	Fdatasync(fd int) (err error)
//sys	Flock(fd int, how int) (err error)
//sys	Fsync(fd int) (err error)
//sys	Getdents(fd int, buf []byte) (n int, err error) = SYS_GETDENTS64
//sysnb	Getpgid(pid int) (pgid int, err error)

func Getpgrp() (pid int) {
	pid, _ = Getpgid(0)
	return
}

//sysnb	Getpid() (pid int)
//sysnb	Getppid() (ppid int)
//sys	Getpriority(which int, who int) (prio int, err error)
//sys	Getrandom(buf []byte, flags int) (n int, err error)
//sysnb	Getrusage(who int, rusage *Rusage) (err error)
//sysnb	Getsid(pid int) (sid int, err error)
//sysnb	Gettid() (tid int)
//sys	Getxattr(path string, attr string, dest []byte) (sz int, err error)
//sys	InotifyAddWatch(fd int, pathname string, mask uint32) (watchdesc int, err error)
//sysnb	InotifyInit1(flags int) (fd int, err error)
//sysnb	InotifyRmWatch(fd int, watchdesc uint32) (success int, err error)
//sysnb	Kill(pid int, sig syscall.Signal) (err error)
//sys	Klogctl(typ int, buf []byte) (n int, err error) = SYS_SYSLOG
//sys	Listxattr(path string, dest []byte) (sz int, err error)
//sys	Mkdirat(dirfd int, path string, mode uint32) (err error)
//sys	Mknodat(dirfd int, path string, mode uint32, dev int) (err error)
//sys	Nanosleep(time *Timespec, leftover *Timespec) (err error)
//sys	PivotRoot(newroot string, putold string) (err error) = SYS_PIVOT_ROOT
//sysnb prlimit(pid int, resource int, newlimit *Rlimit, old *Rlimit) (err error) = SYS_PRLIMIT64
//sys   Prctl(option int, arg2 uintptr, arg3 uintptr, arg4 uintptr, arg5 uintptr) (err error)
//sys	read(fd int, p []byte) (n int, err error)
//sys	Removexattr(path string, attr string) (err error)
//sys	Renameat(olddirfd int, oldpath string, newdirfd int, newpath string) (err error)
//sys	Setdomainname(p []byte) (err error)
//sys	Sethostname(p []byte) (err error)
//sysnb	Setpgid(pid int, pgid int) (err error)
//sysnb	Setsid() (pid int, err error)
//sysnb	Settimeofday(tv *Timeval) (err error)
//sys	Setns(fd int, nstype int) (err error)

// issue 1435.
// On linux Setuid and Setgid only affects the current thread, not the process.
// This does not match what most callers expect so we must return an error
// here rather than letting the caller think that the call succeeded.

func Setuid(uid int) (err error) {
	return EOPNOTSUPP
}

func Setgid(uid int) (err error) {
	return EOPNOTSUPP
}

//sys	Setpriority(which int, who int, prio int) (err error)
//sys	Setxattr(path string, attr string, data []byte, flags int) (err error)
//sys	Sync()
//sysnb	Sysinfo(info *Sysinfo_t) (err error)
//sys	Tee(rfd int, wfd int, len int, flags int) (n int64, err error)
//sysnb	Tgkill(tgid int, tid int, sig syscall.Signal) (err error)
//sysnb	Times(tms *Tms) (ticks uintptr, err error)
//sysnb	Umask(mask int) (oldmask int)
//sysnb	Uname(buf *Utsname) (err error)
//sys	Unmount(target string, flags int) (err error) = SYS_UMOUNT2
//sys	Unshare(flags int) (err error)
//sys	Ustat(dev int, ubuf *Ustat_t) (err error)
//sys	write(fd int, p []byte) (n int, err error)
//sys	exitThread(code int) (err error) = SYS_EXIT
//sys	readlen(fd int, p *byte, np int) (n int, err error) = SYS_READ
//sys	writelen(fd int, p *byte, np int) (n int, err error) = SYS_WRITE

// mmap varies by architecture; see syscall_linux_*.go.
//sys	munmap(addr uintptr, length uintptr) (err error)

var mapper = &mmapper{
	active: make(map[*byte][]byte),
	mmap:   mmap,
	munmap: munmap,
}

func Mmap(fd int, offset int64, length int, prot int, flags int) (data []byte, err error) {
	return mapper.Mmap(fd, offset, length, prot, flags)
}

func Munmap(b []byte) (err error) {
	return mapper.Munmap(b)
}

//sys	Madvise(b []byte, advice int) (err error)
//sys	Mprotect(b []byte, prot int) (err error)
//sys	Mlock(b []byte) (err error)
//sys	Munlock(b []byte) (err error)
//sys	Mlockall(flags int) (err error)
//sys	Munlockall() (err error)

// Vmsplice splices user pages from a slice of Iovecs into a pipe specified by fd,
// using the specified flags.
func Vmsplice(fd int, iovs []Iovec, flags int) (int, error) {
	n, _, errno := Syscall6(
		SYS_VMSPLICE,
		uintptr(fd),
		uintptr(unsafe.Pointer(&iovs[0])),
		uintptr(len(iovs)),
		uintptr(flags),
		0,
		0,
	)
	if errno != 0 {
		return 0, syscall.Errno(errno)
	}

	return int(n), nil
}

/*
 * Unimplemented
 */
// AddKey
// AfsSyscall
// Alarm
// ArchPrctl
// Brk
// Capget
// Capset
// ClockGetres
// ClockNanosleep
// ClockSettime
// Clone
// CreateModule
// DeleteModule
// EpollCtlOld
// EpollPwait
// EpollWaitOld
// Eventfd
// Execve
// Fgetxattr
// Flistxattr
// Fork
// Fremovexattr
// Fsetxattr
// Futex
// GetKernelSyms
// GetMempolicy
// GetRobustList
// GetThreadArea
// Getitimer
// Getpmsg
// IoCancel
// IoDestroy
// IoGetevents
// IoSetup
// IoSubmit
// Ioctl
// IoprioGet
// IoprioSet
// KexecLoad
// Keyctl
// Lgetxattr
// Llistxattr
// LookupDcookie
// Lremovexattr
// Lsetxattr
// Mbind
// MigratePages
// Mincore
// ModifyLdt
// Mount
// MovePages
// Mprotect
// MqGetsetattr
// MqNotify
// MqOpen
// MqTimedreceive
// MqTimedsend
// MqUnlink
// Mremap
// Msgctl
// Msgget
// Msgrcv
// Msgsnd
// Msync
// Newfstatat
// Nfsservctl
// Personality
// Pselect6
// Ptrace
// Putpmsg
// QueryModule
// Quotactl
// Readahead
// Readv
// RemapFilePages
// RequestKey
// RestartSyscall
// RtSigaction
// RtSigpending
// RtSigprocmask
// RtSigqueueinfo
// RtSigreturn
// RtSigsuspend
// RtSigtimedwait
// SchedGetPriorityMax
// SchedGetPriorityMin
// SchedGetaffinity
// SchedGetparam
// SchedGetscheduler
// SchedRrGetInterval
// SchedSetaffinity
// SchedSetparam
// SchedYield
// Security
// Semctl
// Semget
// Semop
// Semtimedop
// SetMempolicy
// SetRobustList
// SetThreadArea
// SetTidAddress
// Shmat
// Shmctl
// Shmdt
// Shmget
// Sigaltstack
// Signalfd
// Swapoff
// Swapon
// Sysfs
// TimerCreate
// TimerDelete
// TimerGetoverrun
// TimerGettime
// TimerSettime
// Timerfd
// Tkill (obsolete)
// Tuxcall
// Umount2
// Uselib
// Utimensat
// Vfork
// Vhangup
// Vserver
// Waitid
// _Sysctl
