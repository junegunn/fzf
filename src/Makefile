UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SUFFIX := darwin
else ifeq ($(UNAME_S),Linux)
	SUFFIX := linux
endif

UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	SUFFIX := $(SUFFIX)_amd64
else ifneq ($(filter i386 i686,$(UNAME_M)),)
	SUFFIX := $(SUFFIX)_386
else # TODO
$(error "$(UNAME_M) is not supported, yet.")
endif

BINARY  := fzf-$(SUFFIX)
BINDIR  := ../bin
SOURCES := $(wildcard *.go fzf/*.go)
RELEASE  = fzf-$(shell fzf/$(BINARY) --version)-$(SUFFIX)

all: release

release: build
	cd fzf && \
	cp $(BINARY) $(RELEASE) && \
	tar -czf $(RELEASE).tgz $(RELEASE) && \
	rm $(RELEASE)

build: fzf/$(BINARY)

fzf/$(BINARY): $(SOURCES)
	go get
	go test -v
	cd fzf && go build -o $(BINARY)

install: fzf/$(BINARY)
	mkdir -p $(BINDIR)
	cp -f fzf/$(BINARY) $(BINDIR)/fzf

clean:
	cd fzf && rm -f $(BINARY) $(RELEASE).tgz

docker:
	docker build -t junegunn/ubuntu-sandbox .

linux64:
	docker run -i -t -u jg -v $(shell cd ..; pwd):/fzf junegunn/ubuntu-sandbox \
		/bin/bash -ci 'cd ~jg/go/src/github.com/junegunn/fzf/src; make'

.PHONY: build release install linux64 clean docker run
