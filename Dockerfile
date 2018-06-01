FROM archlinux/base:latest
RUN pacman -Sy && pacman --noconfirm -S awk tar git curl tmux zsh fish ruby procps
RUN gem install --no-ri --no-rdoc minitest
RUN echo '. /usr/share/bash-completion/completions/git' >> ~/.bashrc
RUN echo '. ~/.bashrc' >> ~/.bash_profile

# Do not set default PS1
RUN rm -f /etc/bash.bashrc
COPY . /fzf
RUN /fzf/install --all
CMD tmux new 'ruby /fzf/test/test_go.rb > out && touch ok' && cat out && [ -e ok ]
