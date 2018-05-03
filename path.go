package mailz // import "github.com/mndrix/mailz"
import (
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var pathRx = regexp.MustCompile(`^(?:(.+)/)?(cur|new|tmp)/(.+):2,([A-Za-z]*)$`)

var ErrInvalidMessagePath = errors.New("Invalid message path")

type Path struct {
	Prefix string
	State  string
	Unique string
	Flags  map[rune]bool
}

type ByFlag []rune

func (f ByFlag) Len() int           { return len(f) }
func (f ByFlag) Less(i, j int) bool { return f[i] < f[j] }
func (f ByFlag) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

func ParsePath(path string) (*Path, error) {
	matches := pathRx.FindStringSubmatch(path)
	if matches == nil {
		return nil, ErrInvalidMessagePath
	}

	p := &Path{
		Prefix: matches[1],
		State:  matches[2],
		Unique: matches[3],
	}
	p.Flags = make(map[rune]bool)
	for _, flag := range matches[4] {
		p.Flags[flag] = true
	}
	return p, nil
}

func (p *Path) FlagString() string {
	flags := make([]rune, 0, len(p.Flags))
	for flag, ok := range p.Flags {
		if ok {
			flags = append(flags, flag)
		}
	}
	sort.Sort(ByFlag(flags))
	return string(flags)
}

func (p *Path) String() string {
	name := strings.Join([]string{
		p.Unique,
		":2,",
		p.FlagString(),
	}, "")
	return filepath.Join(p.Prefix, p.State, name)
}

func (p *Path) ClearFlag(flag rune) {
	delete(p.Flags, flag)
}

func (p *Path) SetFlag(flag rune) {
	p.Flags[flag] = true
}
