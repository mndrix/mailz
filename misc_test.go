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

func TestGenerateUnique(t *testing.T) {
	seen := make(map[string]bool)
	ok := func(u string) {
		if len(seen) < 10 {
			// output a few for inspection
			t.Logf("unique = %s", u)
		}
		if len(u) != 26 {
			t.Errorf("wrong size: %s", u)
		}
		if seen[u] {
			t.Errorf("collision: %s", u)
		}
		seen[u] = true
	}
	for i := 0; i < 1000; i++ {
		u := GenerateUnique()
		ok(u)
	}
}
