MAIL=~/Mail
readonly message_list="mailz-message-list.txt"

# delete a message
d() {
    mailz flags -s T "$1"
}

# display a prompt for input
prompt() {
    printf "\e[0;34m${1}? \e[0m"
}

# list emails
list() {
    generate_list
    render_list
}

generate_list() {
    if [[ (! -d new) || (! -d cur) ]]; then
        summary
        return
    fi

    mailz cur .
    mailz find -c T \
        | xargs mailz head -s Subject -N From -E From -t Received \
        | sort -t "\t" -f -k1 -k4 \
        | awk -F "\t" '
                BEGIN { OFS=FS; ditto="  \"" }
                {
                    subject=$1;
                    date=$4
                }
                {
                    original_subject=subject
                    if (subject==previous_subject)
                        subject=ditto
                    if (length(subject)>60)
                        subject=substr(subject,1,60);
                    if (subject=="")
                        subject="(no subject)";
                    previous_subject=original_subject;
                }
                {
                    if (length($2)>0 && length($2)<length($3))
                        from=$2;
                    else
                        from=$3;
                    original_from=from
                    if (from==previous_from)
                        from=ditto
                    previous_from=original_from
                }
                {
                    cursor=" ";
                    if (FNR==1) cursor=">";
                }
                { print cursor, FNR, subject, from, date; }
              ' \
        > "tmp/${message_list}"
}

render_list() {
    rs -c -z 0 5 <"tmp/${message_list}"
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
summary() {
    (
        cd "${MAIL}"
        mailz count -c T  best better good inbox spam | sed -E 's/	0$//'
    )
}

# select a folder
choose() {
    cd "${MAIL}/$1"
    echo "$1"
    list
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

# remove temporary files
cleanup() {
    echo -n "Cleaning up ..."
    find $MAIL -name "${message_list}" \
        | egrep "/tmp/${message_list}\$" \
        | xargs rm -f
    echo ""
}
trap cleanup EXIT

prompt
while key="$(getkey)"; do
    echo "${key}"
    case $key in
        g)
            prompt 'Which folder'
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
        s) summary ;;
        q) exit ;;
        y) sync -q
           summary
           ;;

        Ctrl-d) exit ;;
        Ctrl-Y) sync ;;
        *) echo "Unknown command: ${key}"
    esac
    prompt
done
