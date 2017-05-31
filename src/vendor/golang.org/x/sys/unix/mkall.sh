#!/usr/bin/env bash
# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# This script runs or (given -n) prints suggested commands to generate files for
# the Architecture/OS specified by the GOARCH and GOOS environment variables.
# See README.md for more information about how the build system works.

GOOSARCH="${GOOS}_${GOARCH}"

# defaults
mksyscall="./mksyscall.pl"
mkerrors="./mkerrors.sh"
zerrors="zerrors_$GOOSARCH.go"
mksysctl=""
zsysctl="zsysctl_$GOOSARCH.go"
mksysnum=
mktypes=
run="sh"
cmd=""

case "$1" in
-syscalls)
	for i in zsyscall*go
	do
		# Run the command line that appears in the first line
		# of the generated file to regenerate it.
		sed 1q $i | sed 's;^// ;;' | sh > _$i && gofmt < _$i > $i
		rm _$i
	done
	exit 0
	;;
-n)
	run="cat"
	cmd="echo"
	shift
esac

case "$#" in
0)
	;;
*)
	echo 'usage: mkall.sh [-n]' 1>&2
	exit 2
esac

if [[ "$GOOS" = "linux" ]] && [[ "$GOARCH" != "sparc64" ]]; then
	# Use then new build system
	# Files generated through docker (use $cmd so you can Ctl-C the build or run)
	$cmd docker build --tag generate:$GOOS $GOOS
	$cmd docker run --interactive --tty --volume $(dirname "$(readlink -f "$0")"):/build generate:$GOOS
	exit
fi

GOOSARCH_in=syscall_$GOOSARCH.go
case "$GOOSARCH" in
_* | *_ | _)
	echo 'undefined $GOOS_$GOARCH:' "$GOOSARCH" 1>&2
	exit 1
	;;
darwin_386)
	mkerrors="$mkerrors -m32"
	mksyscall="./mksyscall.pl -l32"
	mksysnum="./mksysnum_darwin.pl $(xcrun --show-sdk-path --sdk macosx)/usr/include/sys/syscall.h"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
darwin_amd64)
	mkerrors="$mkerrors -m64"
	mksysnum="./mksysnum_darwin.pl $(xcrun --show-sdk-path --sdk macosx)/usr/include/sys/syscall.h"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
darwin_arm)
	mkerrors="$mkerrors"
	mksysnum="./mksysnum_darwin.pl /usr/include/sys/syscall.h"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
darwin_arm64)
	mkerrors="$mkerrors -m64"
	mksysnum="./mksysnum_darwin.pl $(xcrun --show-sdk-path --sdk iphoneos)/usr/include/sys/syscall.h"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
dragonfly_386)
	mkerrors="$mkerrors -m32"
	mksyscall="./mksyscall.pl -l32 -dragonfly"
	mksysnum="curl -s 'http://gitweb.dragonflybsd.org/dragonfly.git/blob_plain/HEAD:/sys/kern/syscalls.master' | ./mksysnum_dragonfly.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
dragonfly_amd64)
	mkerrors="$mkerrors -m64"
	mksyscall="./mksyscall.pl -dragonfly"
	mksysnum="curl -s 'http://gitweb.dragonflybsd.org/dragonfly.git/blob_plain/HEAD:/sys/kern/syscalls.master' | ./mksysnum_dragonfly.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
freebsd_386)
	mkerrors="$mkerrors -m32"
	mksyscall="./mksyscall.pl -l32"
	mksysnum="curl -s 'http://svn.freebsd.org/base/stable/10/sys/kern/syscalls.master' | ./mksysnum_freebsd.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
freebsd_amd64)
	mkerrors="$mkerrors -m64"
	mksysnum="curl -s 'http://svn.freebsd.org/base/stable/10/sys/kern/syscalls.master' | ./mksysnum_freebsd.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
freebsd_arm)
	mkerrors="$mkerrors"
	mksyscall="./mksyscall.pl -l32 -arm"
	mksysnum="curl -s 'http://svn.freebsd.org/base/stable/10/sys/kern/syscalls.master' | ./mksysnum_freebsd.pl"
	# Let the type of C char be signed for making the bare syscall
	# API consistent across over platforms.
	mktypes="GOARCH=$GOARCH go tool cgo -godefs -- -fsigned-char"
	;;
linux_sparc64)
	GOOSARCH_in=syscall_linux_sparc64.go
	unistd_h=/usr/include/sparc64-linux-gnu/asm/unistd.h
	mkerrors="$mkerrors -m64"
	mksysnum="./mksysnum_linux.pl $unistd_h"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
netbsd_386)
	mkerrors="$mkerrors -m32"
	mksyscall="./mksyscall.pl -l32 -netbsd"
	mksysnum="curl -s 'http://cvsweb.netbsd.org/bsdweb.cgi/~checkout~/src/sys/kern/syscalls.master' | ./mksysnum_netbsd.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
netbsd_amd64)
	mkerrors="$mkerrors -m64"
	mksyscall="./mksyscall.pl -netbsd"
	mksysnum="curl -s 'http://cvsweb.netbsd.org/bsdweb.cgi/~checkout~/src/sys/kern/syscalls.master' | ./mksysnum_netbsd.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
openbsd_386)
	mkerrors="$mkerrors -m32"
	mksyscall="./mksyscall.pl -l32 -openbsd"
	mksysctl="./mksysctl_openbsd.pl"
	zsysctl="zsysctl_openbsd.go"
	mksysnum="curl -s 'http://cvsweb.openbsd.org/cgi-bin/cvsweb/~checkout~/src/sys/kern/syscalls.master' | ./mksysnum_openbsd.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
openbsd_amd64)
	mkerrors="$mkerrors -m64"
	mksyscall="./mksyscall.pl -openbsd"
	mksysctl="./mksysctl_openbsd.pl"
	zsysctl="zsysctl_openbsd.go"
	mksysnum="curl -s 'http://cvsweb.openbsd.org/cgi-bin/cvsweb/~checkout~/src/sys/kern/syscalls.master' | ./mksysnum_openbsd.pl"
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
solaris_amd64)
	mksyscall="./mksyscall_solaris.pl"
	mkerrors="$mkerrors -m64"
	mksysnum=
	mktypes="GOARCH=$GOARCH go tool cgo -godefs"
	;;
*)
	echo 'unrecognized $GOOS_$GOARCH: ' "$GOOSARCH" 1>&2
	exit 1
	;;
esac

(
	if [ -n "$mkerrors" ]; then echo "$mkerrors |gofmt >$zerrors"; fi
	case "$GOOS" in
	*)
		syscall_goos="syscall_$GOOS.go"
		case "$GOOS" in
		darwin | dragonfly | freebsd | netbsd | openbsd)
			syscall_goos="syscall_bsd.go $syscall_goos"
			;;
		esac
		if [ -n "$mksyscall" ]; then echo "$mksyscall -tags $GOOS,$GOARCH $syscall_goos $GOOSARCH_in |gofmt >zsyscall_$GOOSARCH.go"; fi
		;;
	esac
	if [ -n "$mksysctl" ]; then echo "$mksysctl |gofmt >$zsysctl"; fi
	if [ -n "$mksysnum" ]; then echo "$mksysnum |gofmt >zsysnum_$GOOSARCH.go"; fi
	if [ -n "$mktypes" ]; then
		echo "$mktypes types_$GOOS.go | go run mkpost.go > ztypes_$GOOSARCH.go";
	fi
) | $run
