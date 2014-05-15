package csvstruct

import (
	"io"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	type row struct {
		Foo, Bar, Baz string
	}

	for _, c := range []struct {
		data string
		out  []row
	}{{
		data: `Foo,Bar,Baz
a,b,c
d,e,f`,
		out: []row{{"a", "b", "c"}, {"d", "e", "f"}},
	}, {
		// Rows that only have partial data are only partially filled.
		data: `Foo,Bar
a,
,b`,
		out: []row{{"a", "", ""}, {"", "b", ""}},
	}, {
		// Disjoint data results in unset struct.
		data: `
Qux
ignored`,
		out: []row{{"", "", ""}},
	}} {
		d := NewDecoder(strings.NewReader(c.data))
		var row row
		i := 0
		for {
			err := d.DecodeNext(&row)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("%v", err)
				break
			}
			if c.out[i] != row {
				t.Errorf("unexpected item %d, got %+v, want %+v", i, row, c.out[i])
			}
			i++
		}
	}
}
