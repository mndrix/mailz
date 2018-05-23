mailz is a command line tool for working with messages in a maildir.
It's designed to be used in shell scripts or combined with standard
Unix tools.

For example, to show the Subject header of all messages that don't
have the T ("trashed") flag:

    mailz find -c T | xargs mailz head -s Subject

To mark a message as seen (S) and replied (R):

    mailz flags -s SR path/to/cur/message:2,

To extract a text version of a message body (decoded, handles MIME):

    mailz body path/to/cur/message:2,

## Installation

    go get -u github.com/mndrix/mailz/...

## Example

I use a shell script built on top of mailz as my main email client.
An early snapshot of that script is in the repository in
[mail-client.sh](examples/mail-client.sh).  It might give you some
ideas of how it can be used.
