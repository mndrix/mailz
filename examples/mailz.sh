# summarize folder contents
alias s="find {good,better,best,inbox,spam}/{cur,new} -type f | awk -F/ '{print \$1}' | sort | uniq -c"


# delete a message
d() {
    mailz set-flags +T "$1"
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

# sync mailstore
sy() {
    if [[ "$1" == "-q" ]]; then
        mbsync gmail
    else 
        mbsync -V -D gmail | less
    fi
}
