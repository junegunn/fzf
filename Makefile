ifndef GOOS
GOOS           := $(word 1, $(subst /, " ", $(word 4, $(shell go version))))
endif

MAKEFILE       := $(realpath $(lastword $(MAKEFILE_LIST)))
ROOT_DIR       := $(shell dirname $(MAKEFILE))
GOPATH         := $(ROOT_DIR)/gopath
SRC_LINK       := $(GOPATH)/src/github.com/junegunn/fzf/src
VENDOR_LINK    := $(GOPATH)/src/github.com/junegunn/fzf/vendor
export GOPATH

GLIDE_YAML     := glide.yaml
GLIDE_LOCK     := glide.lock
SOURCES        := $(wildcard *.go src/*.go src/*/*.go) $(SRC_LINK) $(VENDOR_LINK) $(GLIDE_LOCK) $(MAKEFILE)

REVISION       := $(shell git log -n 1 --pretty=format:%h -- $(SOURCES))
BUILD_FLAGS    := -a -ldflags "-X main.revision=$(REVISION) -w -extldflags=$(LDFLAGS)" -tags "$(TAGS)"

BINARY32       := fzf-$(GOOS)_386
BINARY64       := fzf-$(GOOS)_amd64
BINARYARM5     := fzf-$(GOOS)_arm5
BINARYARM6     := fzf-$(GOOS)_arm6
BINARYARM7     := fzf-$(GOOS)_arm7
BINARYARM8     := fzf-$(GOOS)_arm8
BINARYPPC64LE  := fzf-$(GOOS)_ppc64le
VERSION        := $(shell awk -F= '/version =/ {print $$2}' src/constants.go | tr -d "\" ")
RELEASE32      := fzf-$(VERSION)-$(GOOS)_386
RELEASE64      := fzf-$(VERSION)-$(GOOS)_amd64
RELEASEARM5    := fzf-$(VERSION)-$(GOOS)_arm5
RELEASEARM6    := fzf-$(VERSION)-$(GOOS)_arm6
RELEASEARM7    := fzf-$(VERSION)-$(GOOS)_arm7
RELEASEARM8    := fzf-$(VERSION)-$(GOOS)_arm8
RELEASEPPC64LE := fzf-$(VERSION)-$(GOOS)_ppc64le

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
else ifeq ($(UNAME_M),armv8l)
	BINARY := $(BINARYARM8)
else ifeq ($(UNAME_M),ppc64le)
	BINARY := $(BINARYPPC64LE)
else
$(error "Build on $(UNAME_M) is not supported, yet.")
endif

all: target/$(BINARY)

target:
	mkdir -p $@

ifeq ($(GOOS),windows)
release: target/$(BINARY32) target/$(BINARY64)
	cd target && cp -f $(BINARY32) fzf.exe && zip $(RELEASE32).zip fzf.exe
	cd target && cp -f $(BINARY64) fzf.exe && zip $(RELEASE64).zip fzf.exe
	cd target && rm -f fzf.exe
else ifeq ($(GOOS),linux)
release: target/$(BINARY32) target/$(BINARY64) target/$(BINARYARM5) target/$(BINARYARM6) target/$(BINARYARM7) target/$(BINARYARM8) target/$(BINARYPPC64LE)
	cd target && cp -f $(BINARY32) fzf && tar -czf $(RELEASE32).tgz fzf
	cd target && cp -f $(BINARY64) fzf && tar -czf $(RELEASE64).tgz fzf
	cd target && cp -f $(BINARYARM5) fzf && tar -czf $(RELEASEARM5).tgz fzf
	cd target && cp -f $(BINARYARM6) fzf && tar -czf $(RELEASEARM6).tgz fzf
	cd target && cp -f $(BINARYARM7) fzf && tar -czf $(RELEASEARM7).tgz fzf
	cd target && cp -f $(BINARYARM8) fzf && tar -czf $(RELEASEARM8).tgz fzf
	cd target && cp -f $(BINARYPPC64LE) fzf && tar -czf $(RELEASEPPC64LE).tgz fzf
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
	ln -sf $(ROOT_DIR)/src $(SRC_LINK)

$(VENDOR_LINK):
	mkdir -p $(shell dirname $(VENDOR_LINK))
	ln -sf $(ROOT_DIR)/vendor $(VENDOR_LINK)

vendor: $(GLIDE_YAML)
	go get -u github.com/Masterminds/glide && $(GOPATH)/bin/glide install && touch $@

test: $(SOURCES) vendor
	SHELL=/bin/sh GOOS= go test -v -tags "$(TAGS)" \
				github.com/junegunn/fzf/src \
				github.com/junegunn/fzf/src/algo \
				github.com/junegunn/fzf/src/tui \
				github.com/junegunn/fzf/src/util

install: bin/fzf

clean:
	rm -rf target

target/$(BINARY32): $(SOURCES) vendor
	GOARCH=386 go build $(BUILD_FLAGS) -o $@

target/$(BINARY64): $(SOURCES) vendor
	GOARCH=amd64 go build $(BUILD_FLAGS) -o $@

# https://github.com/golang/go/wiki/GoArm
target/$(BINARYARM5): $(SOURCES) vendor
	GOARCH=arm GOARM=5 go build $(BUILD_FLAGS) -o $@

target/$(BINARYARM6): $(SOURCES) vendor
	GOARCH=arm GOARM=6 go build $(BUILD_FLAGS) -o $@

target/$(BINARYARM7): $(SOURCES) vendor
	GOARCH=arm GOARM=7 go build $(BUILD_FLAGS) -o $@

target/$(BINARYARM8): $(SOURCES) vendor
	GOARCH=arm64 go build $(BUILD_FLAGS) -o $@

target/$(BINARYPPC64LE): $(SOURCES) vendor
	GOARCH=ppc64le go build $(BUILD_FLAGS) -o $@

bin/fzf: target/$(BINARY) | bin
	cp -f target/$(BINARY) bin/fzf

docker:
	docker build -t fzf-arch .
	docker run -it fzf-arch tmux

docker-test:
	docker build -t fzf-arch .
	docker run -it fzf-arch

.PHONY: all release release-all test install clean docker docker-test
