# summarize folder contents
alias s="find {good,better,best,inbox,spam}/{cur,new} -type f | awk -F/ '{print \$1}' | sort | uniq -c"

# sync mailstore
alias sy='mbsync -V -D gmail | less'

# delete a message
d() {
    mv "$1" "${1}T"
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

# move a message to another folder
m() {
    readonly hash="$(sha256 -q $1)"
    readonly file="${hash}:2,"
    readonly tmp="$HOME/Mail/$2/tmp/${file}"
    readonly final="$HOME/Mail/$2/cur/${file}"
    cp "$1" "${tmp}" && mv "${tmp}" "${final}" && d "$1"
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
