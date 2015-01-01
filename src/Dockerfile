FROM ubuntu:14.04
MAINTAINER Junegunn Choi <junegunn.c@gmail.com>

# apt-get
RUN apt-get update && apt-get -y upgrade
RUN apt-get install -y --force-yes git vim-nox curl procps sudo \
                                   build-essential libncurses-dev

# Setup jg user with sudo privilege
RUN useradd -s /bin/bash -m jg && echo 'jg:jg' | chpasswd && \
    echo 'jg ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/jg

# Setup dotfiles
USER jg
RUN cd ~ && git clone https://github.com/junegunn/dotfiles.git && \
    dotfiles/install > /dev/null

# Install Go 1.4
RUN cd ~ && curl https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz | tar -xz && \
    mv go go1.4 && \
    echo 'export GOROOT=~/go1.4' >> ~/dotfiles/bashrc-extra && \
    echo 'export PATH=~/go1.4/bin:$PATH' >> ~/dotfiles/bashrc-extra

# Symlink fzf directory
RUN mkdir -p ~jg/go/src/github.com/junegunn && \
    ln -s /fzf ~jg/go/src/github.com/junegunn/fzf

# Volume
VOLUME /fzf

# Default CMD
CMD cd ~jg/go/src/github.com/junegunn/fzf/src && /bin/bash -l

