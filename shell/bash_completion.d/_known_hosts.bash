# Helper function for completing _known_hosts.
# This function performs host completion based on ssh's config and known_hosts
# files, as well as hostnames reported by avahi-browse if
# COMP_KNOWN_HOSTS_WITH_AVAHI is set to a non-empty value.  Also hosts from
# HOSTFILE (compgen -A hostname) are added, unless
# COMP_KNOWN_HOSTS_WITH_HOSTFILE is set to an empty value.
# Usage: _known_hosts_real [OPTIONS] CWORD
# Options:  -a             Use aliases from ssh config files
#           -c             Use `:' suffix
#           -F configfile  Use `configfile' for configuration settings
#           -p PREFIX      Use PREFIX
#           -4             Filter IPv6 addresses from results
#           -6             Filter IPv4 addresses from results
# Return: Completions, starting with CWORD, are added to COMPREPLY[]
_known_hosts_real()
{
    local configfile flag prefix
    local cur curd awkcur user suffix aliases i host ipv4 ipv6
    local -a kh khd config
    local -a toks

    # TODO remove trailing %foo from entries

    local OPTIND=1
    while getopts "ac46F:p:" flag "$@"; do
        case $flag in
            a) aliases='yes' ;;
            c) suffix=':' ;;
            F) configfile=$OPTARG ;;
            p) prefix=$OPTARG ;;
            4) ipv4=1 ;;
            6) ipv6=1 ;;
        esac
    done
    [[ $# -lt $OPTIND ]] && echo "error: $FUNCNAME: missing mandatory argument CWORD"
    cur=${!OPTIND}; let "OPTIND += 1"
    [[ $# -ge $OPTIND ]] && echo "error: $FUNCNAME("$@"): unprocessed arguments:"\
    $(while [[ $# -ge $OPTIND ]]; do printf '%s\n' ${!OPTIND}; shift; done)

    [[ $cur == *@* ]] && user=${cur%@*}@ && cur=${cur#*@}
    kh=()

    # ssh config files
    if [[ -n $configfile ]]; then
        [[ -r $configfile ]] && config+=( "$configfile" )
    else
        for i in /etc/ssh/ssh_config ~/.ssh/config ~/.ssh2/config; do
            [[ -r $i ]] && config+=( "$i" )
        done
    fi

    # "Include" keyword in ssh config files
    for i in "${config[@]}"; do
        _included_ssh_config_files "$i"
    done

    # Known hosts files from configs
    if [[ ${#config[@]} -gt 0 ]]; then
        local OIFS=$IFS IFS=$'\n' j
        local -a tmpkh
        # expand paths (if present) to global and user known hosts files
        # TODO(?): try to make known hosts files with more than one consecutive
        #          spaces in their name work (watch out for ~ expansion
        #          breakage! Alioth#311595)
        tmpkh=( $( awk 'sub("^[ \t]*([Gg][Ll][Oo][Bb][Aa][Ll]|[Uu][Ss][Ee][Rr])[Kk][Nn][Oo][Ww][Nn][Hh][Oo][Ss][Tt][Ss][Ff][Ii][Ll][Ee][ \t]+", "") { print $0 }' "${config[@]}" | sort -u ) )
        IFS=$OIFS
        for i in "${tmpkh[@]}"; do
            # First deal with quoted entries...
            while [[ $i =~ ^([^\"]*)\"([^\"]*)\"(.*)$ ]]; do
                i=${BASH_REMATCH[1]}${BASH_REMATCH[3]}
                j=${BASH_REMATCH[2]}
                __expand_tilde_by_ref j # Eval/expand possible `~' or `~user'
                [[ -r $j ]] && kh+=( "$j" )
            done
            # ...and then the rest.
            for j in $i; do
                __expand_tilde_by_ref j # Eval/expand possible `~' or `~user'
                [[ -r $j ]] && kh+=( "$j" )
            done
        done
    fi

    if [[ -z $configfile ]]; then
        # Global and user known_hosts files
        for i in /etc/ssh/ssh_known_hosts /etc/ssh/ssh_known_hosts2 \
            /etc/known_hosts /etc/known_hosts2 ~/.ssh/known_hosts \
            ~/.ssh/known_hosts2; do
            [[ -r $i ]] && kh+=( "$i" )
        done
        for i in /etc/ssh2/knownhosts ~/.ssh2/hostkeys; do
            [[ -d $i ]] && khd+=( "$i"/*pub )
        done
    fi

    # If we have known_hosts files to use
    if [[ ${#kh[@]} -gt 0 || ${#khd[@]} -gt 0 ]]; then
        # Escape slashes and dots in paths for awk
        awkcur=${cur//\//\\\/}
        awkcur=${awkcur//\./\\\.}
        curd=$awkcur

        if [[ "$awkcur" == [0-9]*[.:]* ]]; then
            # Digits followed by a dot or a colon - just search for that
            awkcur="^$awkcur[.:]*"
        elif [[ "$awkcur" == [0-9]* ]]; then
            # Digits followed by no dot or colon - search for digits followed
            # by a dot or a colon
            awkcur="^$awkcur.*[.:]"
        elif [[ -z $awkcur ]]; then
            # A blank - search for a dot, a colon, or an alpha character
            awkcur="[a-z.:]"
        else
            awkcur="^$awkcur"
        fi

        if [[ ${#kh[@]} -gt 0 ]]; then
            # FS needs to look for a comma separated list
            toks+=( $( awk 'BEGIN {FS=","}
            /^\s*[^|\#]/ {
            sub("^@[^ ]+ +", ""); \
            sub(" .*$", ""); \
            for (i=1; i<=NF; ++i) { \
            sub("^\\[", "", $i); sub("\\](:[0-9]+)?$", "", $i); \
            if ($i !~ /[*?]/ && $i ~ /'"$awkcur"'/) {print $i} \
            }}' "${kh[@]}" 2>/dev/null ) )
        fi
        if [[ ${#khd[@]} -gt 0 ]]; then
            # Needs to look for files called
            # .../.ssh2/key_22_<hostname>.pub
            # dont fork any processes, because in a cluster environment,
            # there can be hundreds of hostkeys
            for i in "${khd[@]}" ; do
                if [[ "$i" == *key_22_$curd*.pub && -r "$i" ]]; then
                    host=${i/#*key_22_/}
                    host=${host/%.pub/}
                    toks+=( $host )
                fi
            done
        fi

        # apply suffix and prefix
        for (( i=0; i < ${#toks[@]}; i++ )); do
            toks[i]=$prefix$user${toks[i]}$suffix
        done
    fi

    # append any available aliases from ssh config files
    if [[ ${#config[@]} -gt 0 && -n "$aliases" ]]; then
        local hosts=$( command sed -ne 's/^[[:blank:]]*[Hh][Oo][Ss][Tt][[:blank:]]\{1,\}\([^#*?%]*\)\(#.*\)\{0,1\}$/\1/p' "${config[@]}" )
        toks+=( $( compgen -P "$prefix$user" \
            -S "$suffix" -W "$hosts" -- "$cur" ) )
    fi

    # This feature is disabled because it does not scale to
    #  larger networks. See:
    # https://bugs.launchpad.net/ubuntu/+source/bash-completion/+bug/510591
    # https://bugs.debian.org/574950

    # Add hosts reported by avahi-browse, if desired and it's available.
    #if [[ ${COMP_KNOWN_HOSTS_WITH_AVAHI:-} ]] && \
        #type avahi-browse &>/dev/null; then
        # The original call to avahi-browse also had "-k", to avoid lookups
        # into avahi's services DB. We don't need the name of the service, and
        # if it contains ";", it may mistify the result. But on Gentoo (at
        # least), -k wasn't available (even if mentioned in the manpage) some
        # time ago, so...
        #toks+=( $( compgen -P "$prefix$user" -S "$suffix" -W \
        #    "$( avahi-browse -cpr _workstation._tcp 2>/dev/null | \
        #        awk -F';' '/^=/ { print $7 }' | sort -u )" -- "$cur" ) )
    #fi

    # Add hosts reported by ruptime.
    toks+=( $( compgen -W \
        "$( ruptime 2>/dev/null | awk '!/^ruptime:/ { print $1 }' )" \
        -- "$cur" ) )

    # Add results of normal hostname completion, unless
    # `COMP_KNOWN_HOSTS_WITH_HOSTFILE' is set to an empty value.
    if [[ -n ${COMP_KNOWN_HOSTS_WITH_HOSTFILE-1} ]]; then
        toks+=(
            $( compgen -A hostname -P "$prefix$user" -S "$suffix" -- "$cur" ) )
    fi

    if [[ $ipv4 ]]; then
        toks=( "${toks[@]/*:*$suffix/}" )
    fi
    if [[ $ipv6 ]]; then
        toks=( "${toks[@]/+([0-9]).+([0-9]).+([0-9]).+([0-9])$suffix/}" )
    fi
    if [[ $ipv4 || $ipv6 ]]; then
        for i in ${!toks[@]}; do
            [[ ${toks[i]} ]] || unset -v toks[i]
        done
    fi

    local fzf="$(__fzfcmd_complete)"
    local matches
    matches=$(
      printf "%s\n" "${toks[@]}" \
      | FZF_DEFAULT_OPTS="--height ${FZF_TMUX_HEIGHT} --reverse $FZF_DEFAULT_OPTS $FZF_COMPLETION_OPTS" \
        $fzf \
      | while read -r item; do printf "%s " "$item"; done \
    )
    matches=${matches% }

    COMPREPLY=( "${matches}" )

    __ltrim_colon_completions "$prefix$user$cur"

    printf '\e[5n'

} # _known_hosts_real()
complete -F _known_hosts traceroute traceroute6 tracepath tracepath6 \
    fping fping6 telnet rsh rlogin ftp dig mtr ssh-installkeys showmount

