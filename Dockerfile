FROM rubylang/ruby:3.4.1-noble
RUN apt-get update -y && apt install -y git make golang zsh fish tmux
RUN gem install --no-document -v 5.22.3 minitest
RUN echo '. /usr/share/bash-completion/completions/git' >> ~/.bashrc
RUN echo '. ~/.bashrc' >> ~/.bash_profile

# Do not set default PS1
RUN rm -f /etc/bash.bashrc
COPY . /fzf
RUN cd /fzf && make install && ./install --all
ENV LANG=C.UTF-8
CMD ["bash", "-ic", "tmux new 'set -o pipefail; ruby /fzf/test/runner.rb | tee out && touch ok' && cat out && [ -e ok ]"]
