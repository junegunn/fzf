// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

/*
Input to cgo -godefs.  See README.md
*/

// +godefs map struct_in_addr [4]byte /* in_addr */
// +godefs map struct_in6_addr [16]byte /* in6_addr */

package unix

/*
#define KERNEL
#include <dirent.h>
#include <fcntl.h>
#include <signal.h>
#include <termios.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/event.h>
#include <sys/mman.h>
#include <sys/mount.h>
#include <sys/param.h>
#include <sys/ptrace.h>
#include <sys/resource.h>
#include <sys/select.h>
#include <sys/signal.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <net/bpf.h>
#include <net/if.h>
#include <net/if_dl.h>
#include <net/route.h>
#include <netinet/in.h>
#include <netinet/icmp6.h>
#include <netinet/tcp.h>

enum {
	sizeofPtr = sizeof(void*),
};

union sockaddr_all {
	struct sockaddr s1;	// this one gets used for fields
	struct sockaddr_in s2;	// these pad it out
	struct sockaddr_in6 s3;
	struct sockaddr_un s4;
	struct sockaddr_dl s5;
};

struct sockaddr_any {
	struct sockaddr addr;
	char pad[sizeof(union sockaddr_all) - sizeof(struct sockaddr)];
};

// This structure is a duplicate of stat on FreeBSD 8-STABLE.
// See /usr/include/sys/stat.h.
struct stat8 {
#undef st_atimespec	st_atim
#undef st_mtimespec	st_mtim
#undef st_ctimespec	st_ctim
#undef st_birthtimespec	st_birthtim
	__dev_t   st_dev;
	ino_t     st_ino;
	mode_t    st_mode;
	nlink_t   st_nlink;
	uid_t     st_uid;
	gid_t     st_gid;
	__dev_t   st_rdev;
#if __BSD_VISIBLE
	struct  timespec st_atimespec;
	struct  timespec st_mtimespec;
	struct  timespec st_ctimespec;
#else
	time_t    st_atime;
	long      __st_atimensec;
	time_t    st_mtime;
	long      __st_mtimensec;
	time_t    st_ctime;
	long      __st_ctimensec;
#endif
	off_t     st_size;
	blkcnt_t st_blocks;
	blksize_t st_blksize;
	fflags_t  st_flags;
	__uint32_t st_gen;
	__int32_t st_lspare;
#if __BSD_VISIBLE
	struct timespec st_birthtimespec;
	unsigned int :(8 / 2) * (16 - (int)sizeof(struct timespec));
	unsigned int :(8 / 2) * (16 - (int)sizeof(struct timespec));
#else
	time_t    st_birthtime;
	long      st_birthtimensec;
	unsigned int :(8 / 2) * (16 - (int)sizeof(struct __timespec));
	unsigned int :(8 / 2) * (16 - (int)sizeof(struct __timespec));
#endif
};

// This structure is a duplicate of if_data on FreeBSD 8-STABLE.
// See /usr/include/net/if.h.
struct if_data8 {
	u_char  ifi_type;
	u_char  ifi_physical;
	u_char  ifi_addrlen;
	u_char  ifi_hdrlen;
	u_char  ifi_link_state;
	u_char  ifi_spare_char1;
	u_char  ifi_spare_char2;
	u_char  ifi_datalen;
	u_long  ifi_mtu;
	u_long  ifi_metric;
	u_long  ifi_baudrate;
	u_long  ifi_ipackets;
	u_long  ifi_ierrors;
	u_long  ifi_opackets;
	u_long  ifi_oerrors;
	u_long  ifi_collisions;
	u_long  ifi_ibytes;
	u_long  ifi_obytes;
	u_long  ifi_imcasts;
	u_long  ifi_omcasts;
	u_long  ifi_iqdrops;
	u_long  ifi_noproto;
	u_long  ifi_hwassist;
	time_t  ifi_epoch;
	struct  timeval ifi_lastchange;
};

// This structure is a duplicate of if_msghdr on FreeBSD 8-STABLE.
// See /usr/include/net/if.h.
struct if_msghdr8 {
	u_short ifm_msglen;
	u_char  ifm_version;
	u_char  ifm_type;
	int     ifm_addrs;
	int     ifm_flags;
	u_short ifm_index;
	struct  if_data8 ifm_data;
};
*/
import "C"

// Machine characteristics; for internal use.

const (
	sizeofPtr      = C.sizeofPtr
	sizeofShort    = C.sizeof_short
	sizeofInt      = C.sizeof_int
	sizeofLong     = C.sizeof_long
	sizeofLongLong = C.sizeof_longlong
)

// Basic types

type (
	_C_short     C.short
	_C_int       C.int
	_C_long      C.long
	_C_long_long C.longlong
)

// Time

type Timespec C.struct_timespec

type Timeval C.struct_timeval

// Processes

type Rusage C.struct_rusage

type Rlimit C.struct_rlimit

type _Gid_t C.gid_t

// Files

const ( // Directory mode bits
	S_IFMT   = C.S_IFMT
	S_IFIFO  = C.S_IFIFO
	S_IFCHR  = C.S_IFCHR
	S_IFDIR  = C.S_IFDIR
	S_IFBLK  = C.S_IFBLK
	S_IFREG  = C.S_IFREG
	S_IFLNK  = C.S_IFLNK
	S_IFSOCK = C.S_IFSOCK
	S_ISUID  = C.S_ISUID
	S_ISGID  = C.S_ISGID
	S_ISVTX  = C.S_ISVTX
	S_IRUSR  = C.S_IRUSR
	S_IWUSR  = C.S_IWUSR
	S_IXUSR  = C.S_IXUSR
)

type Stat_t C.struct_stat8

type Statfs_t C.struct_statfs

type Flock_t C.struct_flock

type Dirent C.struct_dirent

type Fsid C.struct_fsid

// Advice to Fadvise

const (
	FADV_NORMAL     = C.POSIX_FADV_NORMAL
	FADV_RANDOM     = C.POSIX_FADV_RANDOM
	FADV_SEQUENTIAL = C.POSIX_FADV_SEQUENTIAL
	FADV_WILLNEED   = C.POSIX_FADV_WILLNEED
	FADV_DONTNEED   = C.POSIX_FADV_DONTNEED
	FADV_NOREUSE    = C.POSIX_FADV_NOREUSE
)

// Sockets

type RawSockaddrInet4 C.struct_sockaddr_in

type RawSockaddrInet6 C.struct_sockaddr_in6

type RawSockaddrUnix C.struct_sockaddr_un

type RawSockaddrDatalink C.struct_sockaddr_dl

type RawSockaddr C.struct_sockaddr

type RawSockaddrAny C.struct_sockaddr_any

type _Socklen C.socklen_t

type Linger C.struct_linger

type Iovec C.struct_iovec

type IPMreq C.struct_ip_mreq

type IPMreqn C.struct_ip_mreqn

type IPv6Mreq C.struct_ipv6_mreq

type Msghdr C.struct_msghdr

type Cmsghdr C.struct_cmsghdr

type Inet6Pktinfo C.struct_in6_pktinfo

type IPv6MTUInfo C.struct_ip6_mtuinfo

type ICMPv6Filter C.struct_icmp6_filter

const (
	SizeofSockaddrInet4    = C.sizeof_struct_sockaddr_in
	SizeofSockaddrInet6    = C.sizeof_struct_sockaddr_in6
	SizeofSockaddrAny      = C.sizeof_struct_sockaddr_any
	SizeofSockaddrUnix     = C.sizeof_struct_sockaddr_un
	SizeofSockaddrDatalink = C.sizeof_struct_sockaddr_dl
	SizeofLinger           = C.sizeof_struct_linger
	SizeofIPMreq           = C.sizeof_struct_ip_mreq
	SizeofIPMreqn          = C.sizeof_struct_ip_mreqn
	SizeofIPv6Mreq         = C.sizeof_struct_ipv6_mreq
	SizeofMsghdr           = C.sizeof_struct_msghdr
	SizeofCmsghdr          = C.sizeof_struct_cmsghdr
	SizeofInet6Pktinfo     = C.sizeof_struct_in6_pktinfo
	SizeofIPv6MTUInfo      = C.sizeof_struct_ip6_mtuinfo
	SizeofICMPv6Filter     = C.sizeof_struct_icmp6_filter
)

// Ptrace requests

const (
	PTRACE_TRACEME = C.PT_TRACE_ME
	PTRACE_CONT    = C.PT_CONTINUE
	PTRACE_KILL    = C.PT_KILL
)

// Events (kqueue, kevent)

type Kevent_t C.struct_kevent

// Select

type FdSet C.fd_set

// Routing and interface messages

const (
	sizeofIfMsghdr         = C.sizeof_struct_if_msghdr
	SizeofIfMsghdr         = C.sizeof_struct_if_msghdr8
	sizeofIfData           = C.sizeof_struct_if_data
	SizeofIfData           = C.sizeof_struct_if_data8
	SizeofIfaMsghdr        = C.sizeof_struct_ifa_msghdr
	SizeofIfmaMsghdr       = C.sizeof_struct_ifma_msghdr
	SizeofIfAnnounceMsghdr = C.sizeof_struct_if_announcemsghdr
	SizeofRtMsghdr         = C.sizeof_struct_rt_msghdr
	SizeofRtMetrics        = C.sizeof_struct_rt_metrics
)

type ifMsghdr C.struct_if_msghdr

type IfMsghdr C.struct_if_msghdr8

type ifData C.struct_if_data

type IfData C.struct_if_data8

type IfaMsghdr C.struct_ifa_msghdr

type IfmaMsghdr C.struct_ifma_msghdr

type IfAnnounceMsghdr C.struct_if_announcemsghdr

type RtMsghdr C.struct_rt_msghdr

type RtMetrics C.struct_rt_metrics

// Berkeley packet filter

const (
	SizeofBpfVersion    = C.sizeof_struct_bpf_version
	SizeofBpfStat       = C.sizeof_struct_bpf_stat
	SizeofBpfZbuf       = C.sizeof_struct_bpf_zbuf
	SizeofBpfProgram    = C.sizeof_struct_bpf_program
	SizeofBpfInsn       = C.sizeof_struct_bpf_insn
	SizeofBpfHdr        = C.sizeof_struct_bpf_hdr
	SizeofBpfZbufHeader = C.sizeof_struct_bpf_zbuf_header
)

type BpfVersion C.struct_bpf_version

type BpfStat C.struct_bpf_stat

type BpfZbuf C.struct_bpf_zbuf

type BpfProgram C.struct_bpf_program

type BpfInsn C.struct_bpf_insn

type BpfHdr C.struct_bpf_hdr

type BpfZbufHeader C.struct_bpf_zbuf_header

// Terminal handling

type Termios C.struct_termios
