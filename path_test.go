package mailz // import "github.com/mndrix/mailz"
import "testing"

func TestFlagString(t *testing.T) {
	tests := [][]string{
		{`cur/1525290638.35577_1.x1,U=30:2,`, ``},
		{`cur/1525290638.35577_2.x1,U=31:2,T`, `T`},
		{`cur/foo:2,TSRF`, `FRST`},
	}

	for _, test := range tests {
		path, err := ParsePath(test[0])
		if err != nil {
			t.Errorf("can't parse %q: %s", test[0], err)
			continue
		}

		expected := test[1]
		got := path.FlagString()
		if got != expected {
			t.Errorf("%q != %q", got, expected)
		}
	}
}

func TestCur(t *testing.T) {
	tests := [][]string{
		{`new/foo:2,`, `cur/foo:2,`},
		{`spam/new/baz:2,`, `spam/cur/baz:2,`},
	}

	for _, test := range tests {
		path, err := ParsePath(test[0])
		if err != nil {
			t.Errorf("can't parse %q: %s", test[0], err)
			continue
		}

		expected := test[1]
		path.Cur()
		got := path.String()
		if got != expected {
			t.Errorf("%q != %q", got, expected)
		}
	}
}
