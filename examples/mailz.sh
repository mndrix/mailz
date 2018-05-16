set -e
MAIL=~/Mail
readonly message_list="mailz-message-list.txt"

choose_a_folder() {
    local folder
    prompt 'Which folder'
    local key="$(getkey)"
    case $key in
        g) folder=good ;;
        i) folder=inbox ;;
        p) folder=spam ;;
        s) folder=best ;;
        t) folder=better ;;
        *)
            echo "Unknown folder: ${key}"
            return 1
            ;;
    esac
    echo $folder >&2
    echo $folder
}

compose_new_message() {
    local message="$(mktemp $MAIL/mailz-XXXXXXX)"
    local from="$(from_line)"
    { echo "From: ${from}";
      echo "Bcc: ${from}";
      echo "To: ";
      echo "Subject: ";
    } >>"${message}"
    if "${EDITOR:-vi}" "${message}"; then
        if [[ -s "${message}" ]]; then
            sendmail -v -t <"${message}"
        else
            echo "Aborting. Empty message" >&2
        fi
    fi
    rm -f "${message}"
}

# output the content which should occur on the From: line
from_line() {
    sed -nE '/^set +from=/{ s/^[^"]*"//; s/"$//; p; q; }' ~/.mailrc
}

# display a prompt for input
prompt() {
    printf "\e[0;34m${1}? \e[0m" >/dev/tty
}

# list emails
list() {
    if [[ -d new && -d cur ]]; then
        if [[ ! -e "tmp/${message_list}" ]]; then
            generate_list >"tmp/${message_list}"
        fi
        render_list <"tmp/${message_list}"
    else
        echo "Choose a folder first"
    fi
}

generate_list() {
    mailz cur .
    mailz find -c T \
        | xargs mailz head -i -s Subject -N From -E From -t Received -f \
        | sort -t "$(printf '\t')" -f -k 2 -k 5 \
        | awk '
                BEGIN { FS=OFS="\t" }
                {
                    id=$1;
                    subject=$2;
                    name=$3;
                    email=$4;
                    date=$5;
                    flags=$6;

                    # choose shortest version of From
                    if (length(name)>0 && length(name)<length(email))
                        from=name;
                    else
                        from=email;

                    # indicate the selected message
                    cursor=" ";
                    if (FNR==1) cursor=">";

                    print cursor, id, FNR, subject, from, date, flags;
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
            flags=$7;

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

            print cursor, number, flags, subject, from, date;
        }
    ' \
    | rs -c -z 0 6 \
    | "${PAGER:-more}"
}

# move a message to another folder
move_message() {
    local dst="$(choose_a_folder)"
    if [[ $? != 0 ]]; then
        echo "Trouble selecting a target folder" >&2
        return 1
    fi

    local msg="$(selected_message)"
    if [[ $? != 0 ]]; then
        echo "No message selected" >&2
        return 1
    fi

    local path="$(mailz move ${msg} ../${dst})"

    # remove mbsync's X-TUID header from the copy
    sed -i -E '1,/^$/{ /^X-TUID: /d; }' "${path}"

    if move_cursor "+"; then
        show_selected_line
    fi
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

# move messages from Inbox to Good, Better or Best folders
organize_inbox() {
    cd "${MAIL}/inbox"
    readonly tmp="tmp/mailz-organize-inbox.txt"
    mailz find -c T \
        | xargs mailz head -i -s subject -E from -s list-id \
        | awk '
            BEGIN { FS=OFS="\t" }
            {
                id=$1;
                subject=$2;
                from=$3;
                list=$4;
                if (list=="") list="none";
            }
            function its(folder) {
                print id, folder, subject, from, list;
            }

            # low priority
            from~/chase.com$/ { its("good"); next; }
            from~/google.com$/ && from~/noreply/ { its("good"); next; }
            from~/stripe.com$/ { its("good"); next; }

            # emails from myself
            from~/^michael@ndrix.(org|com)$/ { its("done"); next; }

            # default priority
            { its("better"); }
        ' \
        | tee "${tmp}" \
        | rs -c -z 0 5
    prompt 'Move these messages'
    local response="$(getkey)"
    echo "${response}"
    if [[ $response != "y" ]]; then
        echo "Aborting organization"
        rm -f "${tmp}"
        cd - >/dev/null
        return
    fi

    # move messages
    awk -F t '$2!="done"{print $1, "../" $2}' <"${tmp}" \
        | xargs -n2 mailz move \
        | xargs sed -i -E '1,/^$/{ /^X-TUID: /d; }'

    # mark messages as done
    awk -F t '$2=="done"{print $1}' <"${tmp}" \
        | xargs mailz flags -s T

    # clean up
    rm -f "${tmp}"
    cd - >/dev/null
    summary
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
                mailz body "${path}";
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
        mailz count -c T inbox best better good spam trash | sed -E 's/	0$//'
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
    rm -f "tmp/${message_list}"
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
        c) compose_new_message ;;
        d)
            mark_message_as_done
            if move_cursor "+"; then
                show_selected_line
            fi
            ;;
        g)
            select_folder "$(choose_a_folder)"
            ;;
        l) list ;;
        L)
            rm -f "tmp/${message_list}"
            list
            ;;
        O) organize_inbox ;;
        p) print standard "$(selected_message)" ;;
        P) print verbose "$(selected_message)" ;;
        s) summary ;;
        U) unsubscribe "$(selected_message)" ;;
        v) move_message ;;
        q) exit ;;
        y) sync -q
           summary
           ;;

        Ctrl-d) exit ;;
        Ctrl-n)
            if move_cursor "+"; then
               show_selected_line
            fi
            ;;
        Ctrl-p)
            if move_cursor "-"; then
               show_selected_line
            fi
            ;;
        Ctrl-Y) sync ;;
        *) echo "Unknown command: ${key}"
    esac
    prompt
done
