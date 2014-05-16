package csvstruct

import (
	"fmt"
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
a,"",""
"",b,""
`,
		out: []row{{"a", "", ""}, {"", "b", ""}},
	}, {
		// Rows that don't define all the columns are partially filled.
		data: `Foo,Bar
a,""
"",b
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
		if !isDone(d) {
			t.Errorf("decoder unexpectedly not done")
		}
	}
}

func TestDecode_Unexported(t *testing.T) {
	type row struct {
		Exported, unexported string
	}
	var r row
	if err := NewDecoder(strings.NewReader(`Exported,unexported
a,b`)).DecodeNext(&r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := row{Exported: "a"}
	if !reflect.DeepEqual(r, exp) {
		t.Errorf("unexpected result, got %v, want %v", r, exp)
	}
}

func TestDecode_Tags(t *testing.T) {
	type row struct {
		Foo     string `csv:"renamed_foo"`
		Bar     string
		Ignored string `csv:"-"`
	}
	var r row
	if err := NewDecoder(strings.NewReader(`renamed_foo,Bar,Ignored
a,b,c`)).DecodeNext(&r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := row{"a", "b", ""}
	if !reflect.DeepEqual(r, exp) {
		t.Errorf("unexpected results, got %v, want %v", r, exp)
	}
}

func TestDecode_NonStrings(t *testing.T) {
	type row struct {
		Int     int
		Int64   int64
		Uint64  uint64
		Float64 float64
		Bool    bool
	}
	var r row
	if err := NewDecoder(strings.NewReader(`Int,Int64,Uint64,Float64,Bool
123,-123456789,123456789,123.456,true`)).DecodeNext(&r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := row{123, -123456789, 123456789, 123.456, true}
	if !reflect.DeepEqual(r, exp) {
		t.Errorf("unexpected results, got %v, want %v", r, exp)
	}
}

func TestDecode_Pointers(t *testing.T) {
	t.Skip("pointers are not yet supported")
	type row struct {
		S  string
		SP *string
	}
	var r row
	if err := NewDecoder(strings.NewReader(`S,SP
a,b`)).DecodeNext(&r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if r.S != "a" || r.SP == nil || *r.SP != "b" {
		t.Errorf("unexpected results, got %v", r)
	}
}

func TestDecode_DecodeNil(t *testing.T) {
	type row struct {
		Foo, Bar string
	}
	d := NewDecoder(strings.NewReader(`Foo,Bar
ignore,this
a,b`))
	if err := d.DecodeNext(nil); err != nil {
		t.Errorf("unexpected error while skipping line: %v", err)
	}
	var r row
	if err := d.DecodeNext(&r); err != nil {
		t.Errorf("unexpected error decoding after skip: %v", err)
	}
	exp := row{"a", "b"}
	if r != exp {
		t.Errorf("unexpected result, got %v, want %v", r, exp)
	}
	if !isDone(d) {
		t.Errorf("decoder unexpectedly not done")
	}
}

func isDone(d Decoder) bool {
	return d.DecodeNext(nil) == io.EOF
}

func ExampleDecoder_DecodeNext() {
	csv := `Foo,Bar,Baz
a,b,c
d,e,f`
	type row struct {
		Foo, Bar, Baz string
	}
	var r row
	d := NewDecoder(strings.NewReader(csv))
	for {
		if err := d.DecodeNext(&r); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		fmt.Println(r)
	}
	// Output:
	// {a b c}
	// {d e f}
}
