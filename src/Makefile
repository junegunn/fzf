ifndef GOOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
else
$(error "$$GOOS is not defined.")
endif
endif

SOURCES     := $(wildcard *.go */*.go)
ROOTDIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
BINDIR      := $(shell dirname $(ROOTDIR))/bin
GOPATH      := $(shell dirname $(ROOTDIR))/gopath
SRCDIR      := $(GOPATH)/src/github.com/junegunn/fzf/src
BINARY32    := fzf-$(GOOS)_386
BINARY64    := fzf-$(GOOS)_amd64
BINARYARM5  := fzf-$(GOOS)_arm5
BINARYARM6  := fzf-$(GOOS)_arm6
BINARYARM7  := fzf-$(GOOS)_arm7
BINARYARM8  := fzf-$(GOOS)_arm8
VERSION     := $(shell awk -F= '/version =/ {print $$2}' constants.go | tr -d "\" ")
RELEASE32   := fzf-$(VERSION)-$(GOOS)_386
RELEASE64   := fzf-$(VERSION)-$(GOOS)_amd64
RELEASEARM5 := fzf-$(VERSION)-$(GOOS)_arm5
RELEASEARM6 := fzf-$(VERSION)-$(GOOS)_arm6
RELEASEARM7 := fzf-$(VERSION)-$(GOOS)_arm7
RELEASEARM8 := fzf-$(VERSION)-$(GOOS)_arm8
export GOPATH

# https://en.wikipedia.org/wiki/Uname
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),amd64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),i686)
	BINARY := $(BINARY32)
else ifeq ($(UNAME_M),i386)
	BINARY := $(BINARY32)
else ifeq ($(UNAME_M),armv5l)
	BINARY := $(BINARYARM5)
else ifeq ($(UNAME_M),armv6l)
	BINARY := $(BINARYARM6)
else ifeq ($(UNAME_M),armv7l)
	BINARY := $(BINARYARM7)
else
$(error "Build on $(UNAME_M) is not supported, yet.")
endif

all: fzf/$(BINARY)

ifeq ($(GOOS),windows)
release: fzf/$(BINARY32) fzf/$(BINARY64)
	cd fzf && cp -f $(BINARY32) fzf.exe && zip $(RELEASE32).zip fzf.exe
	cd fzf && cp -f $(BINARY64) fzf.exe && zip $(RELEASE64).zip fzf.exe
	cd fzf && rm -f fzf.exe
else ifeq ($(GOOS),linux)
release: fzf/$(BINARY32) fzf/$(BINARY64) fzf/$(BINARYARM5) fzf/$(BINARYARM6) fzf/$(BINARYARM7) fzf/$(BINARYARM8)
	cd fzf && cp -f $(BINARY32) fzf && tar -czf $(RELEASE32).tgz fzf
	cd fzf && cp -f $(BINARY64) fzf && tar -czf $(RELEASE64).tgz fzf
	cd fzf && cp -f $(BINARYARM5) fzf && tar -czf $(RELEASEARM5).tgz fzf
	cd fzf && cp -f $(BINARYARM6) fzf && tar -czf $(RELEASEARM6).tgz fzf
	cd fzf && cp -f $(BINARYARM7) fzf && tar -czf $(RELEASEARM7).tgz fzf
	cd fzf && cp -f $(BINARYARM8) fzf && tar -czf $(RELEASEARM8).tgz fzf
	cd fzf && rm -f fzf
else
release: fzf/$(BINARY32) fzf/$(BINARY64)
	cd fzf && cp -f $(BINARY32) fzf && tar -czf $(RELEASE32).tgz fzf
	cd fzf && cp -f $(BINARY64) fzf && tar -czf $(RELEASE64).tgz fzf
	cd fzf && rm -f fzf
endif

release-all: clean test
	GOOS=darwin  make release
	GOOS=linux   make release
	GOOS=freebsd make release
	GOOS=openbsd make release
	GOOS=windows make release

$(SRCDIR):
	mkdir -p $(shell dirname $(SRCDIR))
	ln -s $(ROOTDIR) $(SRCDIR)

deps: $(SRCDIR) $(SOURCES)
	cd $(SRCDIR) && go get -tags "$(TAGS)"
	./deps

test: deps
	SHELL=/bin/sh GOOS= go test -v -tags "$(TAGS)" ./...

install: $(BINDIR)/fzf

uninstall:
	rm -f $(BINDIR)/fzf $(BINDIR)/$(BINARY)

clean:
	cd fzf && rm -f fzf-*

fzf/$(BINARY32): deps
	cd fzf && GOARCH=386 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARY32)

fzf/$(BINARY64): deps
	cd fzf && GOARCH=amd64 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARY64)

# https://github.com/golang/go/wiki/GoArm
fzf/$(BINARYARM5): deps
	cd fzf && GOARCH=arm GOARM=5 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARYARM5)

fzf/$(BINARYARM6): deps
	cd fzf && GOARCH=arm GOARM=6 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARYARM6)

fzf/$(BINARYARM7): deps
	cd fzf && GOARCH=arm GOARM=7 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARYARM7)

fzf/$(BINARYARM8): deps
	cd fzf && GOARCH=arm64 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARYARM8)

$(BINDIR)/fzf: fzf/$(BINARY) | $(BINDIR)
	cp -f fzf/$(BINARY) $(BINDIR)
	cd $(BINDIR) && ln -sf $(BINARY) fzf

$(BINDIR):
	mkdir -p $@

.PHONY: all deps release release-all test install uninstall clean
