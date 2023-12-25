SHELL          := bash
GO             ?= go
GOOS           ?= $(word 1, $(subst /, " ", $(word 4, $(shell go version))))

MAKEFILE       := $(realpath $(lastword $(MAKEFILE_LIST)))
ROOT_DIR       := $(shell dirname $(MAKEFILE))
SOURCES        := $(wildcard *.go src/*.go src/*/*.go) $(MAKEFILE)

ifdef FZF_VERSION
VERSION        := $(FZF_VERSION)
else
VERSION        := $(shell git describe --abbrev=0 2> /dev/null)
endif
ifeq ($(VERSION),)
$(error Not on git repository; cannot determine $$FZF_VERSION)
endif
VERSION_TRIM   := $(shell sed "s/-.*//" <<< $(VERSION))
VERSION_REGEX  := $(subst .,\.,$(VERSION_TRIM))

ifdef FZF_REVISION
REVISION       := $(FZF_REVISION)
else
REVISION       := $(shell git log -n 1 --pretty=format:%h --abbrev=8 -- $(SOURCES) 2> /dev/null)
endif
ifeq ($(REVISION),)
$(error Not on git repository; cannot determine $$FZF_REVISION)
endif
BUILD_FLAGS    := -a -ldflags "-s -w -X main.version=$(VERSION) -X main.revision=$(REVISION)" -tags "$(TAGS)"

BINARY32       := fzf-$(GOOS)_386
BINARY64       := fzf-$(GOOS)_amd64
BINARYS390     := fzf-$(GOOS)_s390x
BINARYARM5     := fzf-$(GOOS)_arm5
BINARYARM6     := fzf-$(GOOS)_arm6
BINARYARM7     := fzf-$(GOOS)_arm7
BINARYARM8     := fzf-$(GOOS)_arm8
BINARYPPC64LE  := fzf-$(GOOS)_ppc64le
BINARYRISCV64  := fzf-$(GOOS)_riscv64
BINARYLOONG64  := fzf-$(GOOS)_loong64

# https://en.wikipedia.org/wiki/Uname
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),amd64)
	BINARY := $(BINARY64)
else ifeq ($(UNAME_M),s390x)
	BINARY := $(BINARYS390)
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
else ifeq ($(UNAME_M),armv8l)
	# armv8l is always 32-bit and should implement the armv7 ISA, so
	# just use the same filename as for armv7.
	BINARY := $(BINARYARM7)
else ifeq ($(UNAME_M),arm64)
	BINARY := $(BINARYARM8)
else ifeq ($(UNAME_M),aarch64)
	BINARY := $(BINARYARM8)
else ifeq ($(UNAME_M),ppc64le)
	BINARY := $(BINARYPPC64LE)
else ifeq ($(UNAME_M),riscv64)
	BINARY := $(BINARYRISCV64)
else ifeq ($(UNAME_M),loongarch64)
	BINARY := $(BINARYLOONG64)
else
$(error Build on $(UNAME_M) is not supported, yet.)
endif

all: target/$(BINARY)

test: $(SOURCES)
	[ -z "$$(gofmt -s -d src)" ] || (gofmt -s -d src; exit 1)
	SHELL=/bin/sh GOOS= $(GO) test -v -tags "$(TAGS)" \
				github.com/junegunn/fzf/src \
				github.com/junegunn/fzf/src/algo \
				github.com/junegunn/fzf/src/tui \
				github.com/junegunn/fzf/src/util

bench:
	cd src && SHELL=/bin/sh GOOS= $(GO) test -v -tags "$(TAGS)" -run=Bench -bench=. -benchmem

install: bin/fzf

generate:
	PATH=$(PATH):$(GOPATH)/bin $(GO) generate ./...

build:
	goreleaser build --rm-dist --snapshot --skip-post-hooks

release:
	# Make sure that the tests pass and the build works
	TAGS=tcell make test
	make test build clean

ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN is not defined)
endif

	# Check if we are on master branch
ifneq ($(shell git symbolic-ref --short HEAD),master)
	$(error Not on master branch)
endif

	# Check if version numbers are properly updated
	grep -q ^$(VERSION_REGEX)$$ CHANGELOG.md
	grep -qF '"fzf $(VERSION_TRIM)"' man/man1/fzf.1
	grep -qF '"fzf $(VERSION_TRIM)"' man/man1/fzf-tmux.1
	grep -qF $(VERSION) install
	grep -qF $(VERSION) install.ps1

	# Make release note out of CHANGELOG.md
	mkdir -p tmp
	sed -n '/^$(VERSION_REGEX)$$/,/^[0-9]/p' CHANGELOG.md | tail -r | \
		sed '1,/^ *$$/d' | tail -r | sed 1,2d | tee tmp/release-note

	# Push to temp branch first so that install scripts always works on master branch
	git checkout -B temp master
	git push origin temp --follow-tags --force

	# Make a GitHub release
	goreleaser --rm-dist --release-notes tmp/release-note

	# Push to master
	git checkout master
	git push origin master

	# Delete temp branch
	git push origin --delete temp

clean:
	$(RM) -r dist target

target/$(BINARY32): $(SOURCES)
	GOARCH=386 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARY64): $(SOURCES)
	GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARYS390): $(SOURCES)
	GOARCH=s390x $(GO) build $(BUILD_FLAGS) -o $@
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

target/$(BINARYRISCV64): $(SOURCES)
	GOARCH=riscv64 $(GO) build $(BUILD_FLAGS) -o $@

target/$(BINARYLOONG64): $(SOURCES)
	GOARCH=loong64 $(GO) build $(BUILD_FLAGS) -o $@

bin/fzf: target/$(BINARY) | bin
	-rm -f bin/fzf
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

.PHONY: all generate build release test bench install clean docker docker-test update
