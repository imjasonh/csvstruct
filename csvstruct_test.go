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
d,e,f
`,
		out: []row{{"a", "b", "c"}, {"d", "e", "f"}},
	}, {
		// Rows that only have partial data are only partially filled.
		data: `Foo,Bar,Baz
a,,
,b,
`,
		out: []row{{"a", "", ""}, {"", "b", ""}},
	}, {
		// Rows that don't define all the columns are partially filled.
		data: `Foo,Bar
a,
,b
`,
		out: []row{{"a", "", ""}, {"", "b", ""}},
	}, {
		// Entirely disjoint columns produce empty structs.
		data: `Qux
d
`,
		out: []row{{}},
	}} {
		d := NewDecoder(strings.NewReader(c.data))
		rows := []row{}
		var r row
		for {
			if err := d.DecodeNext(&r); err == io.EOF {
				break
			} else if err != nil {
				t.Errorf("%v", err)
				break
			}
			rows = append(rows, r)
		}
		if !reflect.DeepEqual(rows, c.out) {
			t.Errorf("unexpected result, got %v, want %v", rows, c.out)
		}
	}
}

func TestDecode_Unexported(t *testing.T) {
	type row struct {
		Exported, unexported string
	}
	d := NewDecoder(strings.NewReader(`Exported,unexported
a,b`))
	var r row
	if err := d.DecodeNext(&r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := row{Exported: "a"}
	if !reflect.DeepEqual(r, exp) {
		t.Errorf("unexpected result, got %v, want %v", r, exp)
	}
}
