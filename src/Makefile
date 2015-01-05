UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
endif

ifneq ($(shell uname -m),x86_64)
$(error "Build on $(UNAME_M) is not supported, yet.")
endif

SOURCES   := $(wildcard *.go fzf/*.go)
BINDIR    := ../bin

BINARY32  := fzf-$(GOOS)_386
BINARY64  := fzf-$(GOOS)_amd64
RELEASE32  = fzf-$(shell fzf/$(BINARY64) --version)-$(GOOS)_386
RELEASE64  = fzf-$(shell fzf/$(BINARY64) --version)-$(GOOS)_amd64

all: release

release: build
	cd fzf && \
	cp $(BINARY32) $(RELEASE32) && tar -czf $(RELEASE32).tgz $(RELEASE32) && \
	cp $(BINARY64) $(RELEASE64) && tar -czf $(RELEASE64).tgz $(RELEASE64) && \
	rm $(RELEASE32) $(RELEASE64)

build: fzf/$(BINARY32) fzf/$(BINARY64)

test: $(SOURCES)
	go get
	go test -v

fzf/$(BINARY32): test
	cd fzf && GOARCH=386 CGO_ENABLED=1 go build -o $(BINARY32)

fzf/$(BINARY64): test
	cd fzf && go build -o $(BINARY64)

install: fzf/$(BINARY64)
	mkdir -p $(BINDIR)
	cp -f fzf/$(BINARY64) $(BINDIR)/fzf

clean:
	cd fzf && rm -f $(BINARY32) $(BINARY64) $(RELEASE32).tgz $(RELEASE64).tgz

# Linux distribution to build fzf on
DISTRO := ubuntu

docker:
	docker build -t junegunn/$(DISTRO)-sandbox - < Dockerfile.$(DISTRO)

linux: docker
	docker run -i -t -v $(shell cd ..; pwd):/fzf junegunn/$(DISTRO)-sandbox \
		/bin/bash -ci 'cd /go/src/github.com/junegunn/fzf/src; make'

$(DISTRO): docker
	docker run -i -t -v $(shell cd ..; pwd):/fzf junegunn/$(DISTRO)-sandbox \
		sh -c 'cd /go/src/github.com/junegunn/fzf/src; /bin/bash'

.PHONY: build release install linux clean docker $(DISTRO)
