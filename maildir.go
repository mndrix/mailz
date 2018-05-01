package mailz // import "github.com/mndrix/mailz"

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Maildir represents a mail storage location in maildir format or a
// directory containing other maildirs.
type Maildir struct {
	// Path is an OS-specific path (see "path/filepath") to this
	// maildir's location in the file system.
	Path string

	// HasMessages is true if this maildir contains messages directly.
	// Otherwise, it's a container for other maildirs.
	HasMessages bool

	// Messages contains all the messages within this maildir.
	//Messages []Message

	// Folders contains all the subfolders within this maildir.
	Folders []*Maildir
}

var errNoMessages = errors.New("No messages in folder")

// Skim reads through a maildir's directory structure to find folders
// and messages.
func (md *Maildir) Skim() error {
	// make sure path exists ...
	stat, err := os.Stat(md.Path)
	if err != nil {
		return errors.Wrap(err, "stat maildir")
	}

	// ... and is a directory
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", md.Path)
	}

	// look for subfolders or messages
	dir, err := os.Open(md.Path)
	if err != nil {
		return errors.Wrap(err, "opening maildir")
	}
	hasCur, hasNew, hasTmp := false, false, false
	for {
		entries, err := dir.Readdir(2) // TODO increase after testing
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "reading maildir entries")
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			switch entry.Name() {
			case "cur":
				hasCur = true
			case "new":
				hasNew = true
			case "tmp":
				hasTmp = true
			default:
				folder := &Maildir{}
				folder.Path = filepath.Join(md.Path, entry.Name())
				md.Folders = append(md.Folders, folder)
			}
		}
	}
	md.HasMessages = hasCur && hasNew && hasTmp

	// done reading this directory
	err = dir.Close()
	if err != nil {
		return errors.Wrap(err, "closing maildir")
	}
	dir = nil

	// skim the sub-folders
	trueFolders := make([]*Maildir, 0, len(md.Folders))
	for i, folder := range md.Folders {
		err := folder.Skim()
		switch err {
		case nil:
			trueFolders = append(trueFolders, md.Folders[i])
		case errNoMessages:
		// ignore this folder
		default:
			return errors.Wrap(err, "skimming subfolders")
		}
	}
	md.Folders = trueFolders

	if !md.HasMessages && len(trueFolders) == 0 {
		return errNoMessages
	}
	return nil
}

// HasAnyMessages returns true if this maildir or any of its folders
// has a message.
func (md *Maildir) HasAnyMessages() bool {
	if md.HasMessages {
		return true
	}
	for _, folder := range md.Folders {
		if folder.HasAnyMessages() {
			return true
		}
	}
	return false
}
