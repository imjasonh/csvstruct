package csvstruct

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
)

var ip = net.IPv4(128, 0, 0, 1)

func TestDecode(t *testing.T) {
	type row struct {
		Foo, Bar, Baz string
	}

	for _, c := range []struct {
		s    string
		want []row
	}{{
		"Foo,Bar,Baz\na,b,c\nd,e,f",
		[]row{{"a", "b", "c"}, {"d", "e", "f"}},
	}, {
		// Rows that only have partial data are only partially filled.
		"Foo,Bar,Baz\na,,\n,b,",
		[]row{{"a", "", ""}, {"", "b", ""}},
	}, {
		// Rows that don't define all the columns are partially filled.
		"Foo,Bar\na,\n,b",
		[]row{{"a", "", ""}, {"", "b", ""}},
	}, {
		// Entirely disjoint columns produce empty structs.
		"Qux\nd",
		[]row{{}},
	}} {
		d := NewDecoder(strings.NewReader(c.s))
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
		if !reflect.DeepEqual(rows, c.want) {
			t.Errorf("DecodeNext(%q): got %v, want %v", c.s, rows, c.want)
		}
		if !isDone(d) {
			t.Errorf("decoder unexpectedly not done")
		}
	}
}

func TestDecode_Unexported(t *testing.T) {
	s := "Exported,unexported\na,b"
	type row struct {
		Exported, unexported string
	}
	var r row
	if err := NewDecoder(strings.NewReader(s)).DecodeNext(&r); err != nil {
		t.Errorf("DecodeNext(%q): %v", s, err)
	}
	want := row{Exported: "a"}
	if r != want {
		t.Errorf("DecodeNext(%q): got %v, want %v", s, r, want)
	}
}

func TestDecode_Tags(t *testing.T) {
	s := "renamed_foo,Bar,Ignored\na,b,c"
	type row struct {
		Foo     string `csv:"renamed_foo"`
		Bar     string
		Ignored string `csv:"-"`
	}
	var r row
	if err := NewDecoder(strings.NewReader(s)).DecodeNext(&r); err != nil {
		t.Errorf("DecodeNext(%q): %v", s, err)
	}
	want := row{"a", "b", ""}
	if r != want {
		t.Errorf("DecodeNext(%q): got %v, want %v", s, r, want)
	}
}

func TestDecode_NonStrings(t *testing.T) {
	s := "Int,Int64,Uint64,Float64,Bool\n123,-123456789,123456789,123.456,true"
	type row struct {
		Int     int
		Int64   int64
		Uint64  uint64
		Float64 float64
		Bool    bool
	}
	var r row
	if err := NewDecoder(strings.NewReader(s)).DecodeNext(&r); err != nil {
		t.Errorf("DecodeNext(%q): %v", s, err)
	}
	want := row{123, -123456789, 123456789, 123.456, true}
	if r != want {
		t.Errorf("DecodeNext(%q): got %v, want %v", s, r, want)
	}
}

func TestDecode_IncompatibleTypes(t *testing.T) {
	// Attempting to parse a string as an int will fail in strconv
	s := "Int\nfoo"
	type row struct {
		Int int
	}
	var r row
	if err := NewDecoder(strings.NewReader(s)).DecodeNext(&r); err.Error() != "error decoding: strconv.ParseInt: parsing \"foo\": invalid syntax" {
		t.Errorf("DecodeNext(%q): %v", s, err)
	}
}

func TestDecode_CompatibleTypes(t *testing.T) {
	// Attempting to parse an int as a string will succeed
	s := "String\n123"
	type row struct {
		String string
	}
	var r row
	if err := NewDecoder(strings.NewReader(s)).DecodeNext(&r); err != nil {
		t.Errorf("DecodeNext(%q): %v", s, err)
	}
	want := row{"123"}
	if r != want {
		t.Errorf("DecodeNext(%q): got %v, want %v", s, r, want)
	}
}

func TestDecode_Pointers(t *testing.T) {
	type row struct {
		S  string
		SP *string `csv:",omitempty"`
	}
	b := "b"
	for _, c := range []struct {
		s    string
		want row
	}{
		{"S,SP\na,b", row{"a", &b}},
		{"S,SP\na,", row{"a", nil}},
	} {
		var r row
		if err := NewDecoder(strings.NewReader(c.s)).DecodeNext(&r); err != nil {
			t.Errorf("DecodeNext(%q): %v", c.s, err)
		}
		if !reflect.DeepEqual(r, c.want) {
			t.Errorf("DecodeNext(%q): got %v, want %v", c.s, r, c.want)
		}
	}
}

func TestDecode_DecodeNil(t *testing.T) {
	s := "Foo,Bar\nignore,this\na,b"
	type row struct {
		Foo, Bar string
	}
	d := NewDecoder(strings.NewReader(s))
	if err := d.DecodeNext(nil); err != nil {
		t.Errorf("unexpected error while skipping line: %v", err)
	}
	var r row
	if err := d.DecodeNext(&r); err != nil {
		t.Errorf("unexpected error decoding after skip: %v", err)
	}
	want := row{"a", "b"}
	if r != want {
		t.Errorf("DecodeNext(%q): got %v, want %v", s, r, want)
	}
	if !isDone(d) {
		t.Errorf("decoder unexpectedly not done")
	}
}

func TestDecode_Opts(t *testing.T) {
	// TODO: test LazyQuotes
	type row struct{ A, B, C string }
	want := []row{{"a", "b", "c"}, {"d", "", "f"}}

	for _, c := range []struct {
		opts DecodeOpts
		s    string
	}{{
		DecodeOpts{Comma: '%'},
		`A%B%C
a%b%c
d%""%f`,
	}, {
		DecodeOpts{Comment: '$'},
		`A,B,C
$comment
a,b,c
$comment
d,"",f
$comment`,
	}, {
		DecodeOpts{TrimLeadingSpace: true},
		"A,B,C\n  a,b,c\n\td,,f",
	}} {
		d := NewDecoder(strings.NewReader(c.s)).Opts(c.opts)
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
		if !reflect.DeepEqual(rows, want) {
			t.Errorf("DecodeNext(%q): got %v, want %v", c.s, rows, want)
		}
		if !isDone(d) {
			t.Errorf("decoder unexpectedly not done")
		}
	}
}

func TestDecode_Map(t *testing.T) {
	s := "foo,bar,baz\na,b,c"
	want := map[string]string{
		"foo": "a",
		"bar": "b",
		"baz": "c",
	}
	got := map[string]string{}
	d := NewDecoder(strings.NewReader(s))
	if err := d.DecodeNext(&got); err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DecodeNext(%q): got %v, want %v", s, got, want)
	}
	if !isDone(d) {
		t.Errorf("decoder unexpectedly not done")
	}
}

func TestDecode_MapErrors(t *testing.T) {
	d := NewDecoder(strings.NewReader("foo,bar\na,b"))

	m := map[string]string{}
	if err := d.DecodeNext(m); err == nil {
		t.Errorf("expected error")
	}

	m1 := map[int]string{}
	if err := d.DecodeNext(m1); err == nil {
		t.Errorf("expected error")
	}

	m2 := map[string]int{}
	if err := d.DecodeNext(m2); err == nil {
		t.Errorf("expected error")
	}
}

// Tests that values that implement encoding.TextUnarshaler are correctly unmarshaled.
func TestDecode_TextUnmarshaler(t *testing.T) {
	s := "N\n128.0.0.1"
	d := NewDecoder(strings.NewReader(s))
	var r struct{ N *net.IP }
	if err := d.DecodeNext(&r); err != nil {
		t.Errorf("DecodeNext(%q): %v", s, err)
	}
	if !ip.Equal(*r.N) {
		t.Errorf("DecodeNext(%q): got %v want %v", s, r.N, ip)
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
