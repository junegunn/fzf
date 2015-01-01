BINARY := fzf/fzf

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	BINARY := $(BINARY)_darwin
else ifeq ($(UNAME_S),Linux)
	BINARY := $(BINARY)_linux
endif

UNAME_M := $(shell uname -m)
ifneq ($(filter i386 i686,$(UNAME_M)),)
$(error "filtered is not supported, yet.")
endif

ifeq ($(UNAME_M),x86_64)
	BINARY := $(BINARY)_amd64
else ifneq ($(filter i386 i686,$(UNAME_M)),)
	BINARY := $(BINARY)_386
else # TODO
$(error "$(UNAME_M) is not supported, yet.")
endif

BINDIR  = ../bin
SOURCES = $(wildcard *.go fzf/*.go)

all: build

build: $(BINARY)

$(BINARY): $(SOURCES)
	go get
	go test -v
	cd fzf && go build -o $(notdir $(BINARY))

install: $(BINARY)
	mkdir -p $(BINDIR)
	cp -f $(BINARY) $(BINDIR)/fzf

clean:
	rm -f $(BINARY)

docker:
	docker build -t junegunn/ubuntu-sandbox .

linux64:
	docker run -i -t -u jg -v $(shell cd ..; pwd):/fzf junegunn/ubuntu-sandbox \
		/bin/bash -ci 'cd ~jg/go/src/github.com/junegunn/fzf/src; make build'

.PHONY: build install linux64 clean docker run
