package mailz // import "github.com/mndrix/mailz"
import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mndrix/rand"
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
		return "", fmt.Errorf("Ref matches zero files: %q", ref)
	case 1:
		return matches[0], nil
	default:
		return "", ErrAmbiguousRef
	}
}

// IsMaildir returns true if path refers to a valid maildir in the
// filesystem.
func IsMaildir(path string) bool {
	// the directory must exist
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		// unexpected error
		panic(err)
	}
	if !stat.IsDir() {
		return false
	}

	// ... and each mandatory subdirectory must exist
	for _, subdir := range []string{"cur", "new", "tmp"} {
		subpath := filepath.Join(path, subdir)
		stat, err = os.Stat(subpath)
		if os.IsNotExist(err) {
			return false
		}
		if err != nil {
			// unexpected error
			panic(err)
		}
		if !stat.IsDir() {
			return false
		}
	}

	return true
}

func CommandCopy(args []string) error {
	if len(args) != 2 {
		return errors.New("Must have exactly 2 arguments")
	}

	// where's the source message?
	ref := args[0]
	resolved, err := Resolve(ref)
	if err != nil {
		return errors.Wrap(err, "resolving source")
	}
	src, err := ParsePath(resolved)
	if err != nil {
		return errors.Wrap(err, "parsing source path")
	}

	// where's the destination?
	dst := args[1]
	if !IsMaildir(dst) {
		return fmt.Errorf("Not a maildir: %s", dst)
	}

	// deliver the message to its new maildir
	msg, err := os.Open(resolved)
	if err != nil {
		return errors.Wrap(err, "opening source message")
	}
	defer msg.Close()
	path, err := Deliver(dst, msg, src.FlagString())
	if err != nil {
		return errors.Wrap(err, "delivering message")
	}
	fmt.Println(path)
	return nil
}

func CommandCount(folders []string) error {
	if len(folders) == 0 {
		return errors.New("Must specify a folder")
	}

	// count messages in a single file system directory
	countDir := func(folder, subdir string) (int, error) {
		path := filepath.Join(folder, subdir)
		dir, err := os.Open(path)
		if err != nil {
			return 0, err
		}
		defer dir.Close()
		count := 0
		for {
			entries, err := dir.Readdir(2) // TODO increase after testing
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, errors.Wrap(err, "reading directory entries")
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				_, err := ParsePath(subdir + "/" + entry.Name())
				if err == nil {
					count++
				}
			}
		}
		return count, nil
	}

	for _, folder := range folders {
		curN, err := countDir(folder, "cur")
		if err != nil {
			return errors.Wrap(err, "Counting cur")
		}
		newN, err := countDir(folder, "new")
		if err != nil {
			return errors.Wrap(err, "Counting new")
		}
		fmt.Printf("%s\t%d\n", folder, curN+newN)
	}

	return nil
}

const alpha32 = `0123456789abcdefghjkmnpqrstuvwxy`

func GenerateUnique() string {
	const size = 26 // 130 bits (2 more than standard UUID)
	chars := make([]rune, size)
	for i := 0; i < size; i++ {
		chars[i] = rune(alpha32[rand.Intn(size)])
	}
	return string(chars)
}

// Deliver writes the content of a new message (msg) with specific
// flags into a destination maildir (dst).  To generate the flags
// string, see Path.FlagString()
func Deliver(dst string, msg io.Reader, flags string) (string, error) {
	name := GenerateUnique() + ":2," + flags

	tmp := filepath.Join(dst, "tmp", name)
	out, err := os.Create(tmp)
	if err != nil {
		return "", errors.Wrap(err, "creating temp file")
	}
	defer out.Close()
	_, err = io.Copy(out, msg)
	if err != nil {
		return "", errors.Wrap(err, "writing new message")
	}

	// move from tmp/ to new/
	final := filepath.Join(dst, "new", name)
	err = os.Rename(tmp, final)
	if err != nil {
		return "", errors.Wrap(err, "renaming new message")
	}
	return final, nil
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

func CommandFind(folders []string) error {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	var clear = fs.String("c", "", `Match when these flags are clear, like "ST"`)
	var set = fs.String("s", "", `Match whene these flags are set, like "ST"`)
	if err := fs.Parse(folders); err != nil {
		return errors.Wrap(err, "parsing command line flags")
	}
	folders = fs.Args()
	if len(folders) == 0 {
		folders = []string{"."}
	}
	whereClear := []rune(*clear)
	whereSet := []rune(*set)

	// iterate messages in a single file system directory
	walkDir := func(subdir string) error {
		dir, err := os.Open(subdir)
		if err != nil {
			return err
		}
		defer dir.Close()
		for {
			entries, err := dir.Readdir(2) // TODO increase after testing
			if err == io.EOF {
				break
			}
			if err != nil {
				return errors.Wrap(err, "reading directory entries")
			}
		ENTRY:
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				p := filepath.ToSlash(filepath.Join(subdir, entry.Name()))
				path, err := ParsePath(p)
				if err != nil {
					continue
				}

				// are flag conditions met?
				for _, flag := range whereClear {
					if !path.IsClear(flag) {
						continue ENTRY
					}
				}
				for _, flag := range whereSet {
					if !path.IsSet(flag) {
						continue ENTRY
					}
				}

				fmt.Println(path)
			}
		}
		return nil
	}

	for _, folder := range folders {
		err := walkDir(filepath.Join(folder, "cur"))
		if err != nil {
			return errors.Wrap(err, "Counting cur")
		}
		err = walkDir(filepath.Join(folder, "new"))
		if err != nil {
			return errors.Wrap(err, "Counting new")
		}
	}

	return nil
}

// CommandFlags changes the flags for each message.  For example,
//
//    mailz flags -s SRT -c F path/to/cur/message
//
// sets the flags S, R, and T while clearing the F flag.
func CommandFlags(args []string) error {
	fs := flag.NewFlagSet("flags", flag.ContinueOnError)
	var clear = fs.String("c", "", `A string of flags to clear, like "ST"`)
	var set = fs.String("s", "", `A string of flags to set, like "ST"`)
	if err := fs.Parse(args); err != nil {
		return errors.Wrap(err, "parsing command line flags")
	}
	plus := []rune(*set)
	minus := []rune(*clear)

	var paths []*Path
	for _, arg := range flag.Args() {
		if arg == "" {
			continue
		}
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
