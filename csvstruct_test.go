package csvstruct

import (
	"io"
	"reflect"
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
		rows := []row{}
		var row row
		for {
			err := d.DecodeNext(&row)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("%v", err)
				break
			}
			rows = append(rows, row)
		}
		if reflect.DeepEqual(rows, c.out) {
			t.Errorf("unexpected result, got %v, want %v", rows, c.out)
		}
	}
}
