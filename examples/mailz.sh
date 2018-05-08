MAIL=~/Mail

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
        rg --with-filename --no-line-number --max-count 2 '^(Subject|From): ' cur
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
    local unique="$(mailz unique $(mailz resolve $1))"
    # TODO 1="$(mailz cur ${unique})"
    mailz set-flags +S "${unique}"
    local path="$(mailz resolve ${unique})"
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
    (
        cd "${MAIL}"
        mailz count best better good inbox spam
    )
}

# select a folder
choose() {
    cd "${MAIL}/$1"
    echo "Selected $1"
    l
}

# sync mailstore
sy() {
    if [[ "$1" == "-q" ]]; then
        mbsync gmail
    else 
        mbsync -V -D gmail | less
    fi
}

prompt="? "
echo -n "${prompt}"
while key="$(getkey)"; do
    echo "${key}"
    case $key in
        g)
            key="$(getkey)"
            case $key in
                g) choose good ;;
                i) choose inbox ;;
                p) choose spam ;;
                s) choose best ;;
                t) choose better ;;
                *) echo "Unknown folder: ${key}" ;;
            esac
            ;;
        l) l ;;
        s) s ;;
        q) exit ;;
        x) ex -q ;;
        y) sy -q
           s
           ;;

        Ctrl-Y) sy ;;
        Ctrl-X) eq ;;
        *) echo "Unknown command: ${key}"
    esac
    echo -n "${prompt}"
done
