set -e
MAIL=~/Mail
readonly message_list="mailz-message-list.txt"

# delete a message
d() {
    mailz flags -s T "$1"
}

# display a prompt for input
prompt() {
    printf "\e[0;34m${1}? \e[0m" >/dev/tty
}

# list emails
list() {
    if [[ -d new && -d cur ]]; then
        generate_list >"tmp/${message_list}"
        render_list <"tmp/${message_list}"
    else
        echo "Choose a folder first"
    fi
}

generate_list() {
    if [[ (! -d new) || (! -d cur) ]]; then
        summary
        return
    fi

    mailz cur .
    mailz find -c T \
        | xargs mailz head -i -s Subject -N From -E From -t Received \
        | sort -t "\t" -f -k1 -k4 \
        | awk '
                BEGIN { FS=OFS="\t" }
                {
                    id=$1;
                    subject=$2;
                    name=$3;
                    email=$4;
                    date=$5;

                    # choose shortest version of From
                    if (length(name)>0 && length(name)<length(email))
                        from=name;
                    else
                        from=email;

                    # indicate the selected message
                    cursor=" ";
                    if (FNR==1) cursor=">";

                    print cursor, id, FNR, subject, from, date;
                }
              '
}

render_list() {
    awk '
        BEGIN { FS=OFS="\t"; ditto="  \"" }
        {
            cursor=$1;
            id=$2;
            number=$3;
            subject=$4;
            from=$5;
            date=$6;

            original_subject=subject;
            if (subject==previous_subject)
                subject=ditto
            if (length(subject)>60)
                subject=substr(subject,1,60);
            if (subject=="")
                subject="(no subject)";
            previous_subject=original_subject;

            original_from=from
            if (from==previous_from)
                from=ditto
            previous_from=original_from

            print cursor, number, subject, from, date;
        }
    ' \
    | rs -c -z 0 5 \
    | "${PAGER:-more}"
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

# mark message as done
mark_message_as_done() {
    local id="$(selected_message)"
    mailz flags -s T "${id}"
}

# unselect the first selected message. execute the given ed
# command. select the message on the resulting line.
move_cursor() {
    ed -s <<EOF "tmp/${message_list}" >/dev/null 2>&1
/^>/
s/^>/ /
$1
s/^ />/
w
EOF
}

# display a message
print() {
    local mode=$1
    local id=$2
    if [[ -z $id ]]; then
        echo "No message selected"
        return
    fi

    mailz cur "${id}"
    mailz flags -s S "${id}"
    local path="$(mailz resolve ${id})"
    case $mode in
        standard)
            {
                perl -n -e '
                    exit if /^$/;
                    print if /^(Cc|Date|From|List-ID|Subject|To|X-Mailgun-Sscore):/i;
                ' "${path}" | sort;
                echo;
                mailz body "${path}" | fmt;
            } | ${PAGER:-more}
            ;;
        verbose)
            ${PAGER:-more} "${path}"
            ;;
    esac
}

# reply to a message
r() {
    echo "From someone@example.com Thu Apr 26 18:30:03 2018" >/tmp/message
    cat "$1" >>/tmp/message
    mail -f /tmp/message
}

# outputs the message ID of the first selected message if any
selected_message() {
    awk -F "\t" '$1==">" { print $2; exit; }' "tmp/${message_list}"
}

# summarize folder contents
summary() {
    (
        cd "${MAIL}"
        mailz count -c T  best better good inbox spam | sed -E 's/	0$//'
    )
}

# unsubscribe from the selected message
unsubscribe() {
    local url="$(unsubscribe_url $1)"
    echo "URL: ${url}"
    prompt "Open"
    local response="$(getkey)"
    case $response in
        y)  echo "yes"
            open 2>/dev/null "${url}"
            ;;
        *) echo "no"
           ;;
    esac
}
unsubscribe_url() {
    mailz head -s List-Unsubscribe $1 \
        | awk -F '[<>, ]+' '
            $2~/https?:/ { print $2; exit; }
            $3~/https?:/ { print $3; exit; }
            { print $2, "\n", $3; exit; }
        '
}

# select a folder
select_folder() {
    cd "${MAIL}/$1"
    echo "$1"
    list
}

show_selected_line() {
    sed -nE '/^>/{p;q;}' "tmp/${message_list}" | render_list
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
        0)
            prompt 'Which message'
            read number
            move_cursor "${number}"
            show_selected_line
            ;;
        [1-9])
            move_cursor ${key}
            show_selected_line
            ;;
        d)
            mark_message_as_done
            move_cursor "+" && show_selected_line
            ;;
        g)
            prompt 'Which folder'
            key="$(getkey)"
            case $key in
                g) select_folder good ;;
                i) select_folder inbox ;;
                p) select_folder spam ;;
                s) select_folder best ;;
                t) select_folder better ;;
                *) echo "Unknown folder: ${key}" ;;
            esac
            ;;
        l) list ;;
        p) print standard "$(selected_message)" ;;
        P) print verbose "$(selected_message)" ;;
        s) summary ;;
        U) unsubscribe "$(selected_message)" ;;
        q) exit ;;
        y) sync -q
           summary
           ;;

        Ctrl-d) exit ;;
        Ctrl-n)
            move_cursor "+"
            show_selected_line
            ;;
        Ctrl-p)
            move_cursor "-"
            show_selected_line
            ;;
        Ctrl-Y) sync ;;
        *) echo "Unknown command: ${key}"
    esac
    prompt
done
