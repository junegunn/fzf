FROM --platform=linux/amd64 ubuntu:22.04
RUN apt-get update -y && apt install -y git make golang zsh fish ruby tmux
RUN gem install --no-document -v 5.14.2 minitest
RUN echo '. /usr/share/bash-completion/completions/git' >> ~/.bashrc
RUN echo '. ~/.bashrc' >> ~/.bash_profile

# Do not set default PS1
RUN rm -f /etc/bash.bashrc
COPY . /fzf
RUN cd /fzf && make install && ./install --all
ENV LANG C.UTF-8
CMD tmux new 'set -o pipefail; ruby /fzf/test/test_go.rb | tee out && touch ok' && cat out && [ -e ok ]
