MAIL=~/Mail

# delete a message
d() {
    mailz flags -s T "$1"
}

# list emails
list() {
    if [[ -d new && -d cur ]]; then
        mailz cur .
        mailz find -c T | xargs mailz head -s Received -s From -s Subject
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
        mailz flags -s T "$1"
}

# view a particular email
p() {
    local unique="$(mailz unique $(mailz resolve $1))"
    # TODO 1="$(mailz cur ${unique})"
    mailz flags -s S "${unique}"
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
        mailz count -c T  best better good inbox spam
    )
}

# select a folder
choose() {
    cd "${MAIL}/$1"
    echo "Selected $1"
    l
}

# sync mailstore
sync() {
    if [[ "$1" == "-q" ]]; then
        mbsync gmail
        mbsync --expunge --noop gmail >/dev/null &
    else 
        mbsync -V -D gmail | less
        mbsync -V -D --expunge --noop gmail >/dev/null &
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
        l) list ;;
        s) s ;;
        q) exit ;;
        y) sync -q
           s
           ;;

        Ctrl-d) exit ;;
        Ctrl-Y) sync ;;
        *) echo "Unknown command: ${key}"
    esac
    echo -n "${prompt}"
done
