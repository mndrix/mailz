package mailz // import "github.com/mndrix/mailz"
import "testing"

func TestUnique(t *testing.T) {
	tests := [][]string{
		// filenames from mbsync/isync
		{`cur/1525290638.35577_1.x1,U=30:2,`, `1525290638.35577_1.x1,U=30`},
		{`cur/1525290638.35577_2.x1,U=31:2,T`, `1525290638.35577_2.x1,U=31`},
	}

	for _, test := range tests {
		expected := test[1]
		got, err := Unique(test[0])
		if err != nil {
			t.Errorf("%q: %s", expected, err)
		}
		if got != expected {
			t.Errorf("%q != %q", expected, got)
		}
	}
}
