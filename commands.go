package mailz // import "github.com/mndrix/mailz"
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Unique returns the unique portion of a message's path.  The
// "unique" is stable across the life of a message even when its flags
// or other metadata change.
//
// No attempt is made to verify that each message file exists.
func Unique(path string) (string, error) {
	p, err := ParsePath(path)
	if err != nil {
		return "", errors.Wrap(err, "parse path")
	}
	return p.Unique, nil
}

var ErrAmbiguousRef = errors.New("Ref matches multiple files")
var ErrNoSuchRef = errors.New("Ref matches zero files")

func Resolve(ref string) (string, error) {
	// ref might be a full path
	_, err := os.Stat(ref)
	if err == nil {
		return ref, nil
	}

	// nope. maybe it's a unique in cur/ or new/
	glob := filepath.Join("cur", ref+":2,*")
	matches, err := filepath.Glob(glob)
	if err != nil {
		panic(err)
	}
	if matches == nil {
		glob = filepath.Join("new", ref+":2,*")
		matches, err = filepath.Glob(glob)
		if err != nil {
			panic(err)
		}
	}

	switch len(matches) {
	case 0:
		return "", ErrNoSuchRef
	case 1:
		return matches[0], nil
	default:
		return "", ErrAmbiguousRef
	}
}

func CommandResolve(refs []string) error {
	for _, ref := range refs {
		if path, err := Resolve(ref); err == nil {
			fmt.Println(path)
		} else {
			fmt.Fprintf(os.Stderr, "%s: %s\n", ref, err)
		}
	}

	return nil

}

// CommandSetFlags changes the flags for each message.  For example,
//
//    mailz set-flags +SRT -F path/to/cur/message
//
// adds the flags S, R, and T while removing the F flag.
func CommandSetFlags(args []string) error {
	// parse arguments
	var plus, minus []rune
	var paths []*Path
	for _, arg := range args {
		if arg == "" {
			continue
		}
		switch arg[0] {
		case '-':
			for _, flag := range arg[1:] {
				minus = append(minus, flag)
			}
		case '+':
			for _, flag := range arg[1:] {
				plus = append(plus, flag)
			}
		default:
			path, err := Resolve(arg)
			if err != nil {
				return errors.Wrap(err, "resolve ref")
			}
			p, err := ParsePath(path)
			if err != nil {
				return errors.Wrap(err, "parse path")
			}
			paths = append(paths, p)
		}
	}

	// calculate new names
	oldPaths := make([]string, len(paths))
	newPaths := make([]string, len(paths))
	for i, path := range paths {
		oldPaths[i] = path.String()
		for _, flag := range minus {
			path.ClearFlag(flag)
		}
		for _, flag := range plus {
			path.SetFlag(flag)
		}
		newPaths[i] = path.String()
	}

	// TODO remove after debugging
	fmt.Printf("would add: %s\n", string(plus))
	fmt.Printf("would remove: %s\n", string(minus))
	fmt.Printf("old paths: %+v\n", oldPaths)
	fmt.Printf("new paths: %+v\n", newPaths)

	// rename files
	for i := range oldPaths {
		err := os.Rename(oldPaths[i], newPaths[i])
		if err != nil {
			return errors.Wrap(err, "renaming")
		}
	}

	return nil
}

// CommandUnique outputs, for each message path, the unique portion of
// the message's path.  See Unique.
func CommandUnique(paths []string) error {
	for _, path := range paths {
		if unique, err := Unique(path); err == nil {
			fmt.Println(unique)
		} else {
			fmt.Fprintf(os.Stderr, "Invalid message filename: %s\n", path)
		}
	}

	return nil
}
