// Copyright 2009,2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// FreeBSD system calls.
// This file is compiled as ordinary Go code,
// but it is also input to mksyscall,
// which parses the //sys lines and generates system call stubs.
// Note that sometimes we use a lowercase //sys name and wrap
// it in our own nicer implementation, either here or in
// syscall_bsd.go or syscall_unix.go.

package unix

import "unsafe"

type SockaddrDatalink struct {
	Len    uint8
	Family uint8
	Index  uint16
	Type   uint8
	Nlen   uint8
	Alen   uint8
	Slen   uint8
	Data   [46]int8
	raw    RawSockaddrDatalink
}

// Translate "kern.hostname" to []_C_int{0,1,2,3}.
func nametomib(name string) (mib []_C_int, err error) {
	const siz = unsafe.Sizeof(mib[0])

	// NOTE(rsc): It seems strange to set the buffer to have
	// size CTL_MAXNAME+2 but use only CTL_MAXNAME
	// as the size.  I don't know why the +2 is here, but the
	// kernel uses +2 for its own implementation of this function.
	// I am scared that if we don't include the +2 here, the kernel
	// will silently write 2 words farther than we specify
	// and we'll get memory corruption.
	var buf [CTL_MAXNAME + 2]_C_int
	n := uintptr(CTL_MAXNAME) * siz

	p := (*byte)(unsafe.Pointer(&buf[0]))
	bytes, err := ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}

	// Magic sysctl: "setting" 0.3 to a string name
	// lets you read back the array of integers form.
	if err = sysctl([]_C_int{0, 3}, p, &n, &bytes[0], uintptr(len(name))); err != nil {
		return nil, err
	}
	return buf[0 : n/siz], nil
}

func direntIno(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(Dirent{}.Fileno), unsafe.Sizeof(Dirent{}.Fileno))
}

func direntReclen(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(Dirent{}.Reclen), unsafe.Sizeof(Dirent{}.Reclen))
}

func direntNamlen(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(Dirent{}.Namlen), unsafe.Sizeof(Dirent{}.Namlen))
}

//sysnb pipe() (r int, w int, err error)

func Pipe(p []int) (err error) {
	if len(p) != 2 {
		return EINVAL
	}
	p[0], p[1], err = pipe()
	return
}

func GetsockoptIPMreqn(fd, level, opt int) (*IPMreqn, error) {
	var value IPMreqn
	vallen := _Socklen(SizeofIPMreqn)
	errno := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, errno
}

func SetsockoptIPMreqn(fd, level, opt int, mreq *IPMreqn) (err error) {
	return setsockopt(fd, level, opt, unsafe.Pointer(mreq), unsafe.Sizeof(*mreq))
}

func Accept4(fd, flags int) (nfd int, sa Sockaddr, err error) {
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

func Getfsstat(buf []Statfs_t, flags int) (n int, err error) {
	var _p0 unsafe.Pointer
	var bufsize uintptr
	if len(buf) > 0 {
		_p0 = unsafe.Pointer(&buf[0])
		bufsize = unsafe.Sizeof(Statfs_t{}) * uintptr(len(buf))
	}
	r0, _, e1 := Syscall(SYS_GETFSSTAT, uintptr(_p0), bufsize, uintptr(flags))
	n = int(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

// Derive extattr namespace and attribute name

func xattrnamespace(fullattr string) (ns int, attr string, err error) {
	s := -1
	for idx, val := range fullattr {
		if val == '.' {
			s = idx
			break
		}
	}

	if s == -1 {
		return -1, "", ENOATTR
	}

	namespace := fullattr[0:s]
	attr = fullattr[s+1:]

	switch namespace {
	case "user":
		return EXTATTR_NAMESPACE_USER, attr, nil
	case "system":
		return EXTATTR_NAMESPACE_SYSTEM, attr, nil
	default:
		return -1, "", ENOATTR
	}
}

func initxattrdest(dest []byte, idx int) (d unsafe.Pointer) {
	if len(dest) > idx {
		return unsafe.Pointer(&dest[idx])
	} else {
		return unsafe.Pointer(_zero)
	}
}

// FreeBSD implements its own syscalls to handle extended attributes

func Getxattr(file string, attr string, dest []byte) (sz int, err error) {
	d := initxattrdest(dest, 0)
	destsize := len(dest)

	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return -1, err
	}

	return ExtattrGetFile(file, nsid, a, uintptr(d), destsize)
}

func Fgetxattr(fd int, attr string, dest []byte) (sz int, err error) {
	d := initxattrdest(dest, 0)
	destsize := len(dest)

	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return -1, err
	}

	return ExtattrGetFd(fd, nsid, a, uintptr(d), destsize)
}

func Lgetxattr(link string, attr string, dest []byte) (sz int, err error) {
	d := initxattrdest(dest, 0)
	destsize := len(dest)

	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return -1, err
	}

	return ExtattrGetLink(link, nsid, a, uintptr(d), destsize)
}

// flags are unused on FreeBSD

func Fsetxattr(fd int, attr string, data []byte, flags int) (err error) {
	d := unsafe.Pointer(&data[0])
	datasiz := len(data)

	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return
	}

	_, err = ExtattrSetFd(fd, nsid, a, uintptr(d), datasiz)
	return
}

func Setxattr(file string, attr string, data []byte, flags int) (err error) {
	d := unsafe.Pointer(&data[0])
	datasiz := len(data)

	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return
	}

	_, err = ExtattrSetFile(file, nsid, a, uintptr(d), datasiz)
	return
}

func Lsetxattr(link string, attr string, data []byte, flags int) (err error) {
	d := unsafe.Pointer(&data[0])
	datasiz := len(data)

	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return
	}

	_, err = ExtattrSetLink(link, nsid, a, uintptr(d), datasiz)
	return
}

func Removexattr(file string, attr string) (err error) {
	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return
	}

	err = ExtattrDeleteFile(file, nsid, a)
	return
}

func Fremovexattr(fd int, attr string) (err error) {
	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return
	}

	err = ExtattrDeleteFd(fd, nsid, a)
	return
}

func Lremovexattr(link string, attr string) (err error) {
	nsid, a, err := xattrnamespace(attr)
	if err != nil {
		return
	}

	err = ExtattrDeleteLink(link, nsid, a)
	return
}

func Listxattr(file string, dest []byte) (sz int, err error) {
	d := initxattrdest(dest, 0)
	destsiz := len(dest)

	// FreeBSD won't allow you to list xattrs from multiple namespaces
	s := 0
	var e error
	for _, nsid := range [...]int{EXTATTR_NAMESPACE_USER, EXTATTR_NAMESPACE_SYSTEM} {
		stmp, e := ExtattrListFile(file, nsid, uintptr(d), destsiz)

		/* Errors accessing system attrs are ignored so that
		 * we can implement the Linux-like behavior of omitting errors that
		 * we don't have read permissions on
		 *
		 * Linux will still error if we ask for user attributes on a file that
		 * we don't have read permissions on, so don't ignore those errors
		 */
		if e != nil && e == EPERM && nsid != EXTATTR_NAMESPACE_USER {
			e = nil
			continue
		} else if e != nil {
			return s, e
		}

		s += stmp
		destsiz -= s
		if destsiz < 0 {
			destsiz = 0
		}
		d = initxattrdest(dest, s)
	}

	return s, e
}

func Flistxattr(fd int, dest []byte) (sz int, err error) {
	d := initxattrdest(dest, 0)
	destsiz := len(dest)

	s := 0
	var e error
	for _, nsid := range [...]int{EXTATTR_NAMESPACE_USER, EXTATTR_NAMESPACE_SYSTEM} {
		stmp, e := ExtattrListFd(fd, nsid, uintptr(d), destsiz)
		if e != nil && e == EPERM && nsid != EXTATTR_NAMESPACE_USER {
			e = nil
			continue
		} else if e != nil {
			return s, e
		}

		s += stmp
		destsiz -= s
		if destsiz < 0 {
			destsiz = 0
		}
		d = initxattrdest(dest, s)
	}

	return s, e
}

func Llistxattr(link string, dest []byte) (sz int, err error) {
	d := initxattrdest(dest, 0)
	destsiz := len(dest)

	s := 0
	var e error
	for _, nsid := range [...]int{EXTATTR_NAMESPACE_USER, EXTATTR_NAMESPACE_SYSTEM} {
		stmp, e := ExtattrListLink(link, nsid, uintptr(d), destsiz)
		if e != nil && e == EPERM && nsid != EXTATTR_NAMESPACE_USER {
			e = nil
			continue
		} else if e != nil {
			return s, e
		}

		s += stmp
		destsiz -= s
		if destsiz < 0 {
			destsiz = 0
		}
		d = initxattrdest(dest, s)
	}

	return s, e
}

/*
 * Exposed directly
 */
//sys	Access(path string, mode uint32) (err error)
//sys	Adjtime(delta *Timeval, olddelta *Timeval) (err error)
//sys	Chdir(path string) (err error)
//sys	Chflags(path string, flags int) (err error)
//sys	Chmod(path string, mode uint32) (err error)
//sys	Chown(path string, uid int, gid int) (err error)
//sys	Chroot(path string) (err error)
//sys	Close(fd int) (err error)
//sys	Dup(fd int) (nfd int, err error)
//sys	Dup2(from int, to int) (err error)
//sys	Exit(code int)
//sys	ExtattrGetFd(fd int, attrnamespace int, attrname string, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrSetFd(fd int, attrnamespace int, attrname string, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrDeleteFd(fd int, attrnamespace int, attrname string) (err error)
//sys	ExtattrListFd(fd int, attrnamespace int, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrGetFile(file string, attrnamespace int, attrname string, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrSetFile(file string, attrnamespace int, attrname string, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrDeleteFile(file string, attrnamespace int, attrname string) (err error)
//sys	ExtattrListFile(file string, attrnamespace int, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrGetLink(link string, attrnamespace int, attrname string, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrSetLink(link string, attrnamespace int, attrname string, data uintptr, nbytes int) (ret int, err error)
//sys	ExtattrDeleteLink(link string, attrnamespace int, attrname string) (err error)
//sys	ExtattrListLink(link string, attrnamespace int, data uintptr, nbytes int) (ret int, err error)
//sys	Fadvise(fd int, offset int64, length int64, advice int) (err error) = SYS_POSIX_FADVISE
//sys	Fchdir(fd int) (err error)
//sys	Fchflags(fd int, flags int) (err error)
//sys	Fchmod(fd int, mode uint32) (err error)
//sys	Fchown(fd int, uid int, gid int) (err error)
//sys	Flock(fd int, how int) (err error)
//sys	Fpathconf(fd int, name int) (val int, err error)
//sys	Fstat(fd int, stat *Stat_t) (err error)
//sys	Fstatfs(fd int, stat *Statfs_t) (err error)
//sys	Fsync(fd int) (err error)
//sys	Ftruncate(fd int, length int64) (err error)
//sys	Getdirentries(fd int, buf []byte, basep *uintptr) (n int, err error)
//sys	Getdtablesize() (size int)
//sysnb	Getegid() (egid int)
//sysnb	Geteuid() (uid int)
//sysnb	Getgid() (gid int)
//sysnb	Getpgid(pid int) (pgid int, err error)
//sysnb	Getpgrp() (pgrp int)
//sysnb	Getpid() (pid int)
//sysnb	Getppid() (ppid int)
//sys	Getpriority(which int, who int) (prio int, err error)
//sysnb	Getrlimit(which int, lim *Rlimit) (err error)
//sysnb	Getrusage(who int, rusage *Rusage) (err error)
//sysnb	Getsid(pid int) (sid int, err error)
//sysnb	Gettimeofday(tv *Timeval) (err error)
//sysnb	Getuid() (uid int)
//sys	Issetugid() (tainted bool)
//sys	Kill(pid int, signum syscall.Signal) (err error)
//sys	Kqueue() (fd int, err error)
//sys	Lchown(path string, uid int, gid int) (err error)
//sys	Link(path string, link string) (err error)
//sys	Listen(s int, backlog int) (err error)
//sys	Lstat(path string, stat *Stat_t) (err error)
//sys	Mkdir(path string, mode uint32) (err error)
//sys	Mkfifo(path string, mode uint32) (err error)
//sys	Mknod(path string, mode uint32, dev int) (err error)
//sys	Mlock(b []byte) (err error)
//sys	Mlockall(flags int) (err error)
//sys	Mprotect(b []byte, prot int) (err error)
//sys	Munlock(b []byte) (err error)
//sys	Munlockall() (err error)
//sys	Nanosleep(time *Timespec, leftover *Timespec) (err error)
//sys	Open(path string, mode int, perm uint32) (fd int, err error)
//sys	Pathconf(path string, name int) (val int, err error)
//sys	Pread(fd int, p []byte, offset int64) (n int, err error)
//sys	Pwrite(fd int, p []byte, offset int64) (n int, err error)
//sys	read(fd int, p []byte) (n int, err error)
//sys	Readlink(path string, buf []byte) (n int, err error)
//sys	Rename(from string, to string) (err error)
//sys	Revoke(path string) (err error)
//sys	Rmdir(path string) (err error)
//sys	Seek(fd int, offset int64, whence int) (newoffset int64, err error) = SYS_LSEEK
//sys	Select(n int, r *FdSet, w *FdSet, e *FdSet, timeout *Timeval) (err error)
//sysnb	Setegid(egid int) (err error)
//sysnb	Seteuid(euid int) (err error)
//sysnb	Setgid(gid int) (err error)
//sys	Setlogin(name string) (err error)
//sysnb	Setpgid(pid int, pgid int) (err error)
//sys	Setpriority(which int, who int, prio int) (err error)
//sysnb	Setregid(rgid int, egid int) (err error)
//sysnb	Setreuid(ruid int, euid int) (err error)
//sysnb	Setresgid(rgid int, egid int, sgid int) (err error)
//sysnb	Setresuid(ruid int, euid int, suid int) (err error)
//sysnb	Setrlimit(which int, lim *Rlimit) (err error)
//sysnb	Setsid() (pid int, err error)
//sysnb	Settimeofday(tp *Timeval) (err error)
//sysnb	Setuid(uid int) (err error)
//sys	Stat(path string, stat *Stat_t) (err error)
//sys	Statfs(path string, stat *Statfs_t) (err error)
//sys	Symlink(path string, link string) (err error)
//sys	Sync() (err error)
//sys	Truncate(path string, length int64) (err error)
//sys	Umask(newmask int) (oldmask int)
//sys	Undelete(path string) (err error)
//sys	Unlink(path string) (err error)
//sys	Unmount(path string, flags int) (err error)
//sys	write(fd int, p []byte) (n int, err error)
//sys   mmap(addr uintptr, length uintptr, prot int, flag int, fd int, pos int64) (ret uintptr, err error)
//sys   munmap(addr uintptr, length uintptr) (err error)
//sys	readlen(fd int, buf *byte, nbuf int) (n int, err error) = SYS_READ
//sys	writelen(fd int, buf *byte, nbuf int) (n int, err error) = SYS_WRITE
//sys	accept4(fd int, rsa *RawSockaddrAny, addrlen *_Socklen, flags int) (nfd int, err error)

/*
 * Unimplemented
 */
// Profil
// Sigaction
// Sigprocmask
// Getlogin
// Sigpending
// Sigaltstack
// Ioctl
// Reboot
// Execve
// Vfork
// Sbrk
// Sstk
// Ovadvise
// Mincore
// Setitimer
// Swapon
// Select
// Sigsuspend
// Readv
// Writev
// Nfssvc
// Getfh
// Quotactl
// Mount
// Csops
// Waitid
// Add_profil
// Kdebug_trace
// Sigreturn
// Mmap
// Mlock
// Munlock
// Atsocket
// Kqueue_from_portset_np
// Kqueue_portset
// Getattrlist
// Setattrlist
// Getdirentriesattr
// Searchfs
// Delete
// Copyfile
// Poll
// Watchevent
// Waitevent
// Modwatch
// Getxattr
// Fgetxattr
// Setxattr
// Fsetxattr
// Removexattr
// Fremovexattr
// Listxattr
// Flistxattr
// Fsctl
// Initgroups
// Posix_spawn
// Nfsclnt
// Fhopen
// Minherit
// Semsys
// Msgsys
// Shmsys
// Semctl
// Semget
// Semop
// Msgctl
// Msgget
// Msgsnd
// Msgrcv
// Shmat
// Shmctl
// Shmdt
// Shmget
// Shm_open
// Shm_unlink
// Sem_open
// Sem_close
// Sem_unlink
// Sem_wait
// Sem_trywait
// Sem_post
// Sem_getvalue
// Sem_init
// Sem_destroy
// Open_extended
// Umask_extended
// Stat_extended
// Lstat_extended
// Fstat_extended
// Chmod_extended
// Fchmod_extended
// Access_extended
// Settid
// Gettid
// Setsgroups
// Getsgroups
// Setwgroups
// Getwgroups
// Mkfifo_extended
// Mkdir_extended
// Identitysvc
// Shared_region_check_np
// Shared_region_map_np
// __pthread_mutex_destroy
// __pthread_mutex_init
// __pthread_mutex_lock
// __pthread_mutex_trylock
// __pthread_mutex_unlock
// __pthread_cond_init
// __pthread_cond_destroy
// __pthread_cond_broadcast
// __pthread_cond_signal
// Setsid_with_pid
// __pthread_cond_timedwait
// Aio_fsync
// Aio_return
// Aio_suspend
// Aio_cancel
// Aio_error
// Aio_read
// Aio_write
// Lio_listio
// __pthread_cond_wait
// Iopolicysys
// Mlockall
// Munlockall
// __pthread_kill
// __pthread_sigmask
// __sigwait
// __disable_threadsignal
// __pthread_markcancel
// __pthread_canceled
// __semwait_signal
// Proc_info
// Stat64_extended
// Lstat64_extended
// Fstat64_extended
// __pthread_chdir
// __pthread_fchdir
// Audit
// Auditon
// Getauid
// Setauid
// Getaudit
// Setaudit
// Getaudit_addr
// Setaudit_addr
// Auditctl
// Bsdthread_create
// Bsdthread_terminate
// Stack_snapshot
// Bsdthread_register
// Workq_open
// Workq_ops
// __mac_execve
// __mac_syscall
// __mac_get_file
// __mac_set_file
// __mac_get_link
// __mac_set_link
// __mac_get_proc
// __mac_set_proc
// __mac_get_fd
// __mac_set_fd
// __mac_get_pid
// __mac_get_lcid
// __mac_get_lctx
// __mac_set_lctx
// Setlcid
// Read_nocancel
// Write_nocancel
// Open_nocancel
// Close_nocancel
// Wait4_nocancel
// Recvmsg_nocancel
// Sendmsg_nocancel
// Recvfrom_nocancel
// Accept_nocancel
// Msync_nocancel
// Fcntl_nocancel
// Select_nocancel
// Fsync_nocancel
// Connect_nocancel
// Sigsuspend_nocancel
// Readv_nocancel
// Writev_nocancel
// Sendto_nocancel
// Pread_nocancel
// Pwrite_nocancel
// Waitid_nocancel
// Poll_nocancel
// Msgsnd_nocancel
// Msgrcv_nocancel
// Sem_wait_nocancel
// Aio_suspend_nocancel
// __sigwait_nocancel
// __semwait_signal_nocancel
// __mac_mount
// __mac_get_mount
// __mac_getfsstat
