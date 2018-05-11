package mailz // import "github.com/mndrix/mailz"
import (
	"flag"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	fs := flag.NewFlagSet("count", flag.ContinueOnError)
	q := &Query{}
	allowQueryArguments(fs, q)
	if err := fs.Parse(folders); err != nil {
		return errors.Wrap(err, "parsing command line flags")
	}
	folders = fs.Args()
	if len(folders) == 0 {
		return errors.New("Must specify a folder")
	}

	for _, folder := range folders {
		q.Root = folder
		count := 0
		err := Find(q, func(p *Path) {
			count++
		})
		fmt.Printf("%s\t%d\n", folder, count)
		if err != nil {
			return err
		}
	}

	return nil
}

func CommandCur(paths []string) error {
	q := &Query{OnlyNew: true}
	errs := make([]error, 0)
	for _, path := range paths {
		q.Root = path
		err := Find(q, func(p *Path) {
			src := p.String()
			p.Cur()
			dst := p.String()
			if src == dst {
				return
			}
			debugf("mv %q %q", src, dst)
			err := os.Rename(src, dst)
			if err != nil {
				errs = append(errs, err)
			}
		})
		if err != nil {
			return err
		}
	}

	if len(errs) != 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		return errors.New("some operations failed")
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

type flagList []rune

func (fs *flagList) String() string {
	sort.Sort(ByFlag(*fs))
	return string(*fs)
}
func (fs *flagList) Set(arg string) error {
	*fs = []rune(arg)
	return nil
}

type Query struct {
	// Root is the top-level directory where the search should begin.
	Root string

	// FlagClear is a slice of flags which must be clear (not set) for
	// a message to match.
	FlagClear flagList

	// FlagSet is a slice of flags which must be set for a message to
	// match.
	FlagSet flagList

	// OnlyNew, when true, matches only newly arrived messages.  That
	// is, messages inside the maildir's new/ directory.
	OnlyNew bool
}

func allowQueryArguments(fs *flag.FlagSet, q *Query) {
	fs.Var(&q.FlagClear, "c", `Match when these flags are clear, like "ST"`)
	fs.Var(&q.FlagSet, "s", `Match when these flags are set, like "ST"`)
	fs.BoolVar(&q.OnlyNew, "N", false, `Match only newly arrived messages`)
}

func CommandFind(folders []string) error {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	query := &Query{}
	allowQueryArguments(fs, query)
	if err := fs.Parse(folders); err != nil {
		return errors.Wrap(err, "parsing command line flags")
	}
	folders = fs.Args()
	if len(folders) == 0 {
		folders = []string{"."}
	}
	for _, folder := range folders {
		query.Root = folder
		err := Find(query, func(path *Path) {
			fmt.Println(path)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func debugf(format string, args ...interface{}) {
	if true {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func Find(query *Query, fn func(*Path)) error {
	// handle a single file system entry
	handleEntry := func(p string, entry os.FileInfo) {
		if entry.IsDir() {
			return
		}
		path, err := ParsePath(p)
		if err != nil {
			return
		}

		// are flag conditions met?
		for _, flag := range query.FlagClear {
			if !path.IsClear(flag) {
				return
			}
		}
		for _, flag := range query.FlagSet {
			if !path.IsSet(flag) {
				return
			}
		}

		fn(path)
	}

	// iterate messages in a single file system directory
	walkDir := func(subdir string) error {
		// are we only searching the new/ directory?
		if query.OnlyNew && filepath.Base(subdir) != "new" {
			debugf("OnlyNew skip: %q", subdir)
			return nil
		}

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
			for _, entry := range entries {
				p := filepath.ToSlash(filepath.Join(subdir, entry.Name()))
				handleEntry(p, entry)
			}
		}
		return nil
	}

	entry, err := os.Stat(query.Root)
	if err != nil {
		return errors.Wrap(err, "root missing")
	}
	if !entry.IsDir() {
		handleEntry(query.Root, entry)
		return nil
	}

	err = walkDir(filepath.Join(query.Root, "cur"))
	if err != nil {
		return errors.Wrap(err, "Counting cur")
	}
	err = walkDir(filepath.Join(query.Root, "new"))
	if err != nil {
		return errors.Wrap(err, "Counting new")
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

type columnSpec struct {
	Name   string
	Filter func(*Path, string, string) string
}

func typeAddress(p *Path, h, v string) string {
	addresses, err := mail.ParseAddressList(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid address: %q\n", v)
		return "Error <error>"
	}
	strs := make([]string, len(addresses))
	for i, address := range addresses {
		strs[i] = address.String()
	}
	return strings.Join(strs, ", ")
}

func typeAddressName(p *Path, h, v string) string {
	addresses, err := mail.ParseAddressList(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid address: %q\n", v)
		return "Error <error>"
	}
	strs := make([]string, len(addresses))
	for i, address := range addresses {
		strs[i] = strings.Replace(address.Name, "\t", "        ", -1)
	}
	return strings.Join(strs, ", ")
}

func typeAddressEmail(p *Path, h, v string) string {
	addresses, err := mail.ParseAddressList(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid address: %q\n", v)
		return "Error <error>"
	}
	strs := make([]string, len(addresses))
	for i, address := range addresses {
		strs[i] = address.Address
	}
	return strings.Join(strs, ", ")
}

func typeString(p *Path, h, v string) string {
	return v
}

func typeTime(p *Path, h, v string) string {
	if strings.ToLower(h) == "received" {
		i := strings.LastIndex(v, ";")
		v = strings.TrimSpace(v[i+1:])
	}
	t, err := mail.ParseDate(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid date: %q\n", v)
	}
	return t.UTC().Format(time.RFC3339)
}

func CommandHead(args []string) error {
	// parse command line arguments
	columns := make([]columnSpec, 0)
	paths := make([]*Path, 0)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-a", "-E", "-N", "-s", "-t":
			i++
			if i >= len(args) {
				return errors.New(arg + " needs an argument")
			}
			column := columnSpec{
				Name: args[i],
			}
			switch arg {
			case "-a":
				column.Filter = typeAddress
			case "-E":
				column.Filter = typeAddressEmail
			case "-N":
				column.Filter = typeAddressName
			case "-s":
				column.Filter = typeString
			case "-t":
				column.Filter = typeTime
			default:
				panic("incomplete case statement")
			}
			columns = append(columns, column)
		default:
			if strings.HasPrefix(arg, "-") {
				return errors.New("invalid argument: " + arg)
			}
			resolved, err := Resolve(arg)
			if err != nil {
				return errors.Wrap(err, "resolving argument")
			}
			path, err := ParsePath(resolved)
			if err != nil {
				return errors.Wrap(err, "parsing path")
			}
			paths = append(paths, path)
		}
	}

	// parse the header from each path
	values := make([]string, len(columns))
	for _, path := range paths {
		r, err := os.Open(path.String())
		if err != nil {
			return errors.Wrap(err, "open")
		}
		msg, err := mail.ReadMessage(r)
		if err != nil {
			return errors.Wrap(err, "reading message")
		}
		r.Close()
		for i, column := range columns {
			raw := msg.Header.Get(column.Name)
			values[i] = column.Filter(path, column.Name, raw)
		}
		fmt.Println(strings.Join(values, "\t"))
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
