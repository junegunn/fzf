FROM archlinux/base:latest
RUN pacman -Sy && pacman --noconfirm -S awk git tmux zsh fish ruby procps go make gcc
RUN gem install --no-document minitest
RUN echo '. /usr/share/bash-completion/completions/git' >> ~/.bashrc
RUN echo '. ~/.bashrc' >> ~/.bash_profile

# Do not set default PS1
RUN rm -f /etc/bash.bashrc
COPY . /fzf
RUN cd /fzf && make install && ./install --all
CMD tmux new 'set -o pipefail; ruby /fzf/test/test_go.rb | tee out && touch ok' && cat out && [ -e ok ]
