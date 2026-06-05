FROM rubylang/ruby:3.4.1-noble
RUN apt-get update && apt-get install -y git make golang zsh fish tmux

# https://www.nushell.sh/book/installation.html
RUN <<EOF
    set -ex
    apt-get install -y wget gnupg
    wget -qO- https://apt.fury.io/nushell/gpg.key | gpg --dearmor -o /etc/apt/keyrings/fury-nushell.gpg
    echo "deb [signed-by=/etc/apt/keyrings/fury-nushell.gpg] https://apt.fury.io/nushell/ /" | tee /etc/apt/sources.list.d/fury-nushell.list
    apt-get update
    apt-get install -y nushell
EOF
RUN gem install --no-document -v 5.22.3 minitest
RUN echo '. /usr/share/bash-completion/completions/git' >> ~/.bashrc
RUN echo '. ~/.bashrc' >> ~/.bash_profile

# Do not set default PS1
RUN rm -f /etc/bash.bashrc
COPY . /fzf
RUN cd /fzf && make install && ./install --all
ENV LANG=C.UTF-8
CMD ["bash", "-ic", "tmux new 'set -o pipefail; ruby /fzf/test/runner.rb | tee out && touch ok' && cat out && [ -e ok ]"]
