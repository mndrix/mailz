package mailz // import "github.com/mndrix/mailz"
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Mailz struct {
	// Maildir represents maildir that's currently being examined.
	Maildir Maildir
}

// Main is the entry point for command line execution of mailz
func Main() {
	m := &Mailz{}

	// handle subcommands
	if len(os.Args) > 1 {
		err := Dispatch(os.Args[1:])
		if err != nil {
			m.Fatal("Error with command %q: %s", os.Args[1], err)
		}
		return
	}

	// TODO start co-processes

	// locate the maildir
	err := m.CallHook("maildir")
	if err != nil {
		m.Fatal("Can't locate maildir: %s", err)
	}
	if m.Maildir.Path == "" {
		m.Fatal("maildir hook did not set maildir-path")
	}

	// skim maildir's directory structure
	err = m.Maildir.Skim()
	if err != nil {
		m.Fatal("invalid maildir %s: %s", m.Maildir.Path, err)
	}
	fmt.Printf("DEBUG: %+v\n", m.Maildir)

	// report that there's no mail
	if !m.Maildir.HasAnyMessages() {
		err = m.CallHook("no-mail")
	}
	fmt.Printf("DEBUG: There's mail but I don't know how to read it yet\n")

	// TODO render message list (to show subfolders)

	return
}

func Dispatch(args []string) error {
	var err error

	switch args[0] {
	case "copy":
		err = CommandCopy(args[1:])
	case "count":
		err = CommandCount(args[1:])
	case "find":
		err = CommandFind(args[1:])
	case "resolve":
		err = CommandResolve(args[1:])
	case "set-flags":
		err = CommandSetFlags(args[1:])
	case "unique":
		err = CommandUnique(args[1:])
	default:
		return fmt.Errorf("Unknown subcommand %q", args[0])
	}

	return err
}

// CallHook invokes a hook with its required parameters
func (m *Mailz) CallHook(name string) error {
	// pretend to be a hook until they're implemented
	//
	// >> indicates values sent from mailz to a hook
	// << indicates a value sent from a hook to mailz

	switch name {
	case "maildir": // register maildir env-HOME
		// >> env-HOME
		home, ok := os.LookupEnv("HOME")
		if !ok {
			return errors.New("No HOME in environment")
		}

		// << set maildir-path /home/foo/Mail
		m.Maildir.Path = filepath.Join(home, "Mail")
		return nil
	case "no-mail": // register no-mail maildir-path
		// >> maildir-path
		// << print "No mail in ..."
		fmt.Printf("No mail in %s\n", m.Maildir.Path)
		return nil
	default:
		return fmt.Errorf("Nobody has registered to handle %s", name)
	}
}

func (m *Mailz) Fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
