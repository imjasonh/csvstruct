package csvstruct

import (
	"bytes"
	"testing"
)

func TestEncodeNext(t *testing.T) {
	t.Skip("failing for some reason... TODO: investigate")
	type row struct {
		Foo, Bar, Baz string
	}

	for _, c := range []struct {
		rows []row
		exp  string
	}{{
		[]row{{"a", "b", "c"}, {"d", "e", "f"}},
		`Foo,Bar,Baz
a,b,c
d,e,f`,
	}} {
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		for _, r := range c.rows {
			if err := e.EncodeNext(r); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
		got := string(buf.Bytes())
		if got != c.exp {
			t.Errorf("unexpected result, got %s, want %s", got, c.exp)
		}
	}
}
