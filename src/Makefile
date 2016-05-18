ifndef GOOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
endif
endif

SOURCES     := $(wildcard *.go */*.go)
ROOTDIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
BINDIR      := $(shell dirname $(ROOTDIR))/bin
GOPATH      := $(shell dirname $(ROOTDIR))/gopath
SRCDIR      := $(GOPATH)/src/github.com/junegunn/fzf/src
DOCKEROPTS  := -i -t -v $(ROOTDIR):/fzf/src
BINARY32    := fzf-$(GOOS)_386
BINARY64    := fzf-$(GOOS)_amd64
BINARYARM7  := fzf-$(GOOS)_arm7
VERSION     := $(shell awk -F= '/version =/ {print $$2}' constants.go | tr -d "\" ")
RELEASE32   := fzf-$(VERSION)-$(GOOS)_386
RELEASE64   := fzf-$(VERSION)-$(GOOS)_amd64
RELEASEARM7 := fzf-$(VERSION)-$(GOOS)_arm7
export GOPATH

UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),i686)
	BINARY := $(BINARY32)
else
$(error "Build on $(UNAME_M) is not supported, yet.")
endif

all: fzf/$(BINARY)

release: test fzf/$(BINARY32) fzf/$(BINARY64)
	-cd fzf && cp $(BINARY32) $(RELEASE32) && tar -czf $(RELEASE32).tgz $(RELEASE32)
	cd fzf && cp $(BINARY64) $(RELEASE64) && tar -czf $(RELEASE64).tgz $(RELEASE64) && \
	rm -f $(RELEASE32) $(RELEASE64)

$(SRCDIR):
	mkdir -p $(shell dirname $(SRCDIR))
	ln -s $(ROOTDIR) $(SRCDIR)

deps: $(SRCDIR) $(SOURCES)
	cd $(SRCDIR) && go get

android-build: $(SRCDIR)
	cd $(SRCDIR) && GOARCH=arm GOARM=7 CGO_ENABLED=1 go get
	cd $(SRCDIR)/fzf && GOARCH=arm GOARM=7 CGO_ENABLED=1 go build -a -ldflags="-w -extldflags=-pie" -o $(BINARYARM7)
	cd $(SRCDIR)/fzf && cp $(BINARYARM7) $(RELEASEARM7) && tar -czf $(RELEASEARM7).tgz $(RELEASEARM7) && \
	rm -f $(RELEASEARM7)

test: deps
	SHELL=/bin/sh go test -v ./...

install: $(BINDIR)/fzf

uninstall:
	rm -f $(BINDIR)/fzf $(BINDIR)/$(BINARY)

clean:
	cd fzf && rm -f fzf-*

fzf/$(BINARY32): deps
	cd fzf && GOARCH=386 CGO_ENABLED=1 go build -a -ldflags -w -tags "$(TAGS)" -o $(BINARY32)

fzf/$(BINARY64): deps
	cd fzf && go build -a -ldflags -w -tags "$(TAGS)" -o $(BINARY64)

$(BINDIR)/fzf: fzf/$(BINARY) | $(BINDIR)
	cp -f fzf/$(BINARY) $(BINDIR)
	cd $(BINDIR) && ln -sf $(BINARY) fzf

$(BINDIR):
	mkdir -p $@

docker-arch:
	docker build -t junegunn/arch-sandbox - < Dockerfile.arch

docker-ubuntu:
	docker build -t junegunn/ubuntu-sandbox - < Dockerfile.ubuntu

docker-centos:
	docker build -t junegunn/centos-sandbox - < Dockerfile.centos

docker-android:
	docker build -t junegunn/android-sandbox - < Dockerfile.android

arch: docker-arch
	docker run $(DOCKEROPTS) junegunn/$@-sandbox \
		sh -c 'cd /fzf/src; /bin/bash'

ubuntu: docker-ubuntu
	docker run $(DOCKEROPTS) junegunn/$@-sandbox \
		sh -c 'cd /fzf/src; /bin/bash'

centos: docker-centos
	docker run $(DOCKEROPTS) junegunn/$@-sandbox \
		sh -c 'cd /fzf/src; /bin/bash'

linux: docker-centos
	docker run $(DOCKEROPTS) junegunn/centos-sandbox \
		/bin/bash -ci 'cd /fzf/src; make TAGS=static release'

ubuntu-android: docker-android
	docker run $(DOCKEROPTS) junegunn/android-sandbox \
		sh -c 'cd /fzf/src; /bin/bash'

android: docker-android
	docker run $(DOCKEROPTS) junegunn/android-sandbox \
		/bin/bash -ci 'cd /fzf/src; GOOS=android make android-build'

.PHONY: all deps release test install uninstall clean \
	linux arch ubuntu centos docker-arch docker-ubuntu docker-centos \
	android-build docker-android ubuntu-android android
