# This is a snapshot of the script I use for reading and responding to
# email.  The workflow may not be useful to anyone else, but it can
# serve as an example of how mailz might be used in shell scripts.
#
# Search for 'mailz' to see it in action.
#
# If you want to try this code for yourself, you'll need to
#
#    go get github.com/mndrix/getkey/cmd/getkey
#
# So the script can read and decode single keystrokes without the user
# having to press Enter.  You might also want to change MAIL so that
# you don't accidentally modify your own mailstore.

set -e
MAIL=~/Mail
readonly message_list="mailz-message-list.txt"
readonly mute_list="${MAIL}/muted.txt"

choose_a_folder() {
    local folder
    prompt 'Which folder'
    local key="$(getkey)"
    case $key in
        g) folder=good ;;
        h) folder=trash ;;
        i) folder=inbox ;;
        p) folder=spam ;;
        s) folder=best ;;
        t) folder=better ;;
        *)
            echo "Unknown folder: ${key}" >&2
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
    edit_and_send_mail "${message}" || true
}

edit_and_send_mail() {
    local message="$1"
    if "${EDITOR:-vi}" "${message}"; then
        if [[ -s "${message}" ]]; then
            sendmail -v -t <"${message}"
            rm -f "${message}"
        else
            echo "Aborting. Empty message" >&2
            rm -f "${message}"
            return 1
        fi
    fi
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
        | perl -ne '@F=split /\t/; $F[1]=~s/^re: +//i; print join("\t",@F);' \
        | sort -t "$(printf '\t')" -f -k 2,2 -k 5,5 \
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

# reply to the selected message
reply_to_message() {
    local id="$(selected_message)"
    if [[ $? != 0 ]]; then
        echo "No message selected" >&2
        return 1
    fi
    local message="$(mktemp $MAIL/mailz-XXXXXXX)"
    local from="$(from_line)"
    local to="$(mailz head -s from ${id})"
    local cc="$(mailz head -s to -s cc ${id} | awk -F t '{print $1 ", " $2}' | sed 's/, $//' )"
    local subject="$(mailz head -s subject ${id} | perl -pe 's/(^re: +)?/Re: /i')"
    local parent="$(mailz head -s message-id ${id})"
    local references="$(mailz head -s references ${id})"
    { echo "From: ${from}";
      echo "Bcc: ${from}";
      echo "In-Reply-To: ${parent}";
      echo "References: ${references} ${parent}";
      echo "To: ${to}";
      echo "Cc: ${cc}";
      echo "Subject: ${subject}";
      echo
      mailz body "${id}" | sed 's/^/> /'
    } >>"${message}"
    if edit_and_send_mail "${message}"; then
        mailz flags -s R "${id}"
    fi
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
    mailz flags -s T "$1"
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

# mute emails whose Subject matches this message's Subject
mute_subject() {
    mailz head -s Subject "$1" | perl -pe 's/^re: +//i' >>"${mute_list}"
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
                mailz head -H -z -F '\n' \
                      -s From -s Subject -s To -s Cc -s Date \
                      -s List-ID -s X-Mailgun-Sscore \
                      "${path}";
                echo;
                mailz body "${path}";
            } | ${PAGER:-more}
            ;;
        verbose)
            ${PAGER:-more} "${path}"
            ;;
    esac
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
            id="$(selected_message)"
            mark_message_as_done "${id}"
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
        m)
            id="$(selected_message)"
            mute_subject "${id}"
            mark_message_as_done "${id}"
            echo "Muted"
            if move_cursor "+"; then
                print standard "$(selected_message)"
            fi
            ;;
        p) print standard "$(selected_message)" ;;
        P) print verbose "$(selected_message)" ;;
        s) summary ;;
        u)
            id="$(selected_message)"
            unsubscribe "${id}"
            mark_message_as_done "${id}"
            ;;
        v) move_message ;;
        q) exit ;;
        r) reply_to_message ;;
        y) sync -q
           summary
           ;;

        Enter)
            mark_message_as_done "$(selected_message)"
            if move_cursor "+"; then
                print standard "$(selected_message)"
            fi
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
