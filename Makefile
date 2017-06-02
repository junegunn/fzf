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

ROOT_DIR    := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GOPATH      := $(ROOT_DIR)/gopath
SRC_LINK    := $(GOPATH)/src/github.com/junegunn/fzf/src
VENDOR_LINK := $(GOPATH)/src/github.com/junegunn/fzf/vendor

GLIDE_YAML  := glide.yaml
GLIDE_LOCK  := glide.lock
SOURCES     := $(wildcard *.go src/*.go src/*/*.go) $(SRC_LINK) $(VENDOR_LINK) $(GLIDE_LOCK)

BINARY32    := fzf-$(GOOS)_386
BINARY64    := fzf-$(GOOS)_amd64
BINARYARM5  := fzf-$(GOOS)_arm5
BINARYARM6  := fzf-$(GOOS)_arm6
BINARYARM7  := fzf-$(GOOS)_arm7
BINARYARM8  := fzf-$(GOOS)_arm8
VERSION     := $(shell awk -F= '/version =/ {print $$2}' src/constants.go | tr -d "\" ")
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

all: target/$(BINARY)

target:
	mkdir -p $@

ifeq ($(GOOS),windows)
release: target/$(BINARY32) target/$(BINARY64)
	cd target && cp -f $(BINARY32) fzf.exe && zip $(RELEASE32).zip bin/fzf.exe
	cd target && cp -f $(BINARY64) fzf.exe && zip $(RELEASE64).zip bin/fzf.exe
	cd target && rm -f fzf.exe
else ifeq ($(GOOS),linux)
release: target/$(BINARY32) target/$(BINARY64) target/$(BINARYARM5) target/$(BINARYARM6) target/$(BINARYARM7) target/$(BINARYARM8)
	cd target && cp -f $(BINARY32) fzf && tar -czf $(RELEASE32).tgz fzf
	cd target && cp -f $(BINARY64) fzf && tar -czf $(RELEASE64).tgz fzf
	cd target && cp -f $(BINARYARM5) fzf && tar -czf $(RELEASEARM5).tgz fzf
	cd target && cp -f $(BINARYARM6) fzf && tar -czf $(RELEASEARM6).tgz fzf
	cd target && cp -f $(BINARYARM7) fzf && tar -czf $(RELEASEARM7).tgz fzf
	cd target && cp -f $(BINARYARM8) fzf && tar -czf $(RELEASEARM8).tgz fzf
	cd target && rm -f fzf
else
release: target/$(BINARY32) target/$(BINARY64)
	cd target && cp -f $(BINARY32) fzf && tar -czf $(RELEASE32).tgz fzf
	cd target && cp -f $(BINARY64) fzf && tar -czf $(RELEASE64).tgz fzf
	cd target && rm -f fzf
endif

release-all: clean test
	GOOS=darwin  make release
	GOOS=linux   make release
	GOOS=freebsd make release
	GOOS=openbsd make release
	GOOS=windows make release

$(SRC_LINK):
	mkdir -p $(shell dirname $(SRC_LINK))
	ln -s $(ROOT_DIR)/src $(SRC_LINK)

$(VENDOR_LINK):
	mkdir -p $(shell dirname $(VENDOR_LINK))
	ln -s $(ROOT_DIR)/vendor $(VENDOR_LINK)

$(GLIDE_LOCK): $(GLIDE_YAML)
	go get -u github.com/Masterminds/glide && $(GOPATH)/bin/glide install && touch $@

test: $(SOURCES)
	SHELL=/bin/sh GOOS= go test -v -tags "$(TAGS)" \
				github.com/junegunn/fzf/src \
				github.com/junegunn/fzf/src/algo \
				github.com/junegunn/fzf/src/tui \
				github.com/junegunn/fzf/src/util

install: bin/fzf

clean:
	rm -rf target

target/$(BINARY32): $(SOURCES)
	GOARCH=386 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $@

target/$(BINARY64): $(SOURCES)
	GOARCH=amd64 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $@

# https://github.com/golang/go/wiki/GoArm
target/$(BINARYARM5): $(SOURCES)
	GOARCH=arm GOARM=5 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $@

target/$(BINARYARM6): $(SOURCES)
	GOARCH=arm GOARM=6 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $@

target/$(BINARYARM7): $(SOURCES)
	GOARCH=arm GOARM=7 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $@

target/$(BINARYARM8): $(SOURCES)
	GOARCH=arm64 go build -a -ldflags "-w -extldflags=$(LDFLAGS)" -tags "$(TAGS)" -o $@

bin/fzf: target/$(BINARY) | bin
	cp -f target/$(BINARY) bin/fzf

.PHONY: all release release-all test install clean
