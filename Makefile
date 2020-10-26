GO             ?= go
GOOS           ?= $(word 1, $(subst /, " ", $(word 4, $(shell go version))))

MAKEFILE       := $(realpath $(lastword $(MAKEFILE_LIST)))
ROOT_DIR       := $(shell dirname $(MAKEFILE))
SOURCES        := $(wildcard *.go src/*.go src/*/*.go) $(MAKEFILE)

VERSION        := $(shell git describe --abbrev=0)
REVISION       := $(shell git log -n 1 --pretty=format:%h -- $(SOURCES))
BUILD_FLAGS    := -a -ldflags "-X main.version=$(VERSION) -X main.revision=$(REVISION) -w '-extldflags=$(LDFLAGS)'" -tags "$(TAGS)"

BINARY64       := fzf-$(GOOS)_amd64
BINARYARM5     := fzf-$(GOOS)_arm5
BINARYARM6     := fzf-$(GOOS)_arm6
BINARYARM7     := fzf-$(GOOS)_arm7
BINARYARM8     := fzf-$(GOOS)_arm8
BINARYPPC64LE  := fzf-$(GOOS)_ppc64le
VERSION        := $(shell awk -F= '/version =/ {print $$2}' src/constants.go | tr -d "\" ")

# https://en.wikipedia.org/wiki/Uname
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),amd64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),armv5l)
	BINARY := $(BINARYARM5)
else ifeq ($(UNAME_M),armv6l)
	BINARY := $(BINARYARM6)
else ifeq ($(UNAME_M),armv7l)
	BINARY := $(BINARYARM7)
else ifeq ($(UNAME_M),armv8l)
	BINARY := $(BINARYARM8)
else ifeq ($(UNAME_M),aarch64)
	BINARY := $(BINARYARM8)
else ifeq ($(UNAME_M),ppc64le)
	BINARY := $(BINARYPPC64LE)
else
$(error "Build on $(UNAME_M) is not supported, yet.")
endif

all: target/$(BINARY)

test: $(SOURCES)
	SHELL=/bin/sh GOOS= $(GO) test -v -tags "$(TAGS)" \
				github.com/junegunn/fzf/src \
				github.com/junegunn/fzf/src/algo \
				github.com/junegunn/fzf/src/tui \
				github.com/junegunn/fzf/src/util

install: bin/fzf

release:
	goreleaser --rm-dist --snapshot

clean:
	$(RM) -r dist target

target/$(BINARY64): $(SOURCES)
	GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $@

# https://github.com/golang/go/wiki/GoArm
target/$(BINARYARM5): $(SOURCES)
	GOARCH=arm GOARM=5 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARYARM6): $(SOURCES)
	GOARCH=arm GOARM=6 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARYARM7): $(SOURCES)
	GOARCH=arm GOARM=7 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARYARM8): $(SOURCES)
	GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARYPPC64LE): $(SOURCES)
	GOARCH=ppc64le $(GO) build $(BUILD_FLAGS) -o $@

bin/fzf: target/$(BINARY) | bin
	cp -f target/$(BINARY) bin/fzf

docker:
	docker build -t fzf-arch .
	docker run -it fzf-arch tmux

docker-test:
	docker build -t fzf-arch .
	docker run -it fzf-arch

update:
	$(GO) get -u
	$(GO) mod tidy

.PHONY: all release test install clean docker docker-test update
