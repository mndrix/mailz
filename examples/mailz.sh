# delete a message
d() {
    mailz set-flags +T "$1"
}

# expunge mailstore
ex() {
    if [[ "$1" == "-q" ]]; then
        mbsync ${opts} --expunge --noop gmail
    else
        mbsync -V -D --expunge --noop gmail
    fi
}

# list emails
l() {
    if [[ -d new && -d cur ]]; then
        mv new/* cur/
        rg --with-filename --no-line-number '^(Subject|From): ' cur
    else
        s
    fi
}

# copy a message to another folder
c() {
    mailz copy "$1" "$2"
}

# move a message to another folder
m() {
    mailz copy "$1" "$2" && \
        mailz set-flags +T "$1"
}

# view a particular email
p() {
    readonly unique="$(mailz unique $1)"
    # TODO 1="$(mailz cur ${unique})"
    mailz set-flags +S "${unique}"
    readonly path="$(mailz resolve ${unique})"
    less -p '^(Subject|From):' "${path}"
}

# reply to a message
r() {
    echo "From someone@example.com Thu Apr 26 18:30:03 2018" >/tmp/message
    cat "$1" >>/tmp/message
    mail -f /tmp/message
}

# summarize folder contents
s() {
    mailz count best better good inbox spam | awk '$2==0{$2=""} {printf "%4s %s\n", $2, $1}'
}

# sync mailstore
sy() {
    if [[ "$1" == "-q" ]]; then
        mbsync gmail
    else 
        mbsync -V -D gmail | less
    fi
}
