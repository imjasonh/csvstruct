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
	if r != exp {
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
	if r != exp {
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
	if r != exp {
		t.Errorf("unexpected results, got %v, want %v", r, exp)
	}
}

func TestDecode_IncompatibleTypes(t *testing.T) {
	// Attempting to parse a string as an int will fail in strconv
	type row struct {
		Int int
	}
	var r row
	if err := NewDecoder(strings.NewReader(`Int
foo`)).DecodeNext(&r); err.Error() != "error decoding: strconv.ParseInt: parsing \"foo\": invalid syntax" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecode_CompatibleTypes(t *testing.T) {
	// Attempting to parse an int as a string will succeed
	type row struct {
		String string
	}
	var r row
	if err := NewDecoder(strings.NewReader(`String
123`)).DecodeNext(&r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if r.String != "123" {
		t.Errorf("unexpected results, got %v, want %v", r, row{"123"})
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

func TestDecode_Opts(t *testing.T) {
	type row struct{ A, B, C string }
	exp := []row{{"a", "b", "c"}, {"d", "", "f"}}

	for _, c := range []struct {
		opts DecoderOpts
		data string
	}{{
		DecoderOpts{Comma: '%'},
		`A%B%C
a%b%c
d%""%f
`,
	}, {
		DecoderOpts{Comment: '$'},
		`A,B,C
$comment
a,b,c
$comment
d,"",f
$comment
`,
	}, {
		DecoderOpts{LazyQuotes: true},
		`A,B,C
a,b,c
d,,f
`,
	}, {
		DecoderOpts{TrimLeadingSpace: true},
		`A,B,C
  a,b,c
	d,"",f
`,
	}} {
		d := NewDecoder(strings.NewReader(c.data)).Opts(c.opts)
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
		if !reflect.DeepEqual(rows, exp) {
			t.Errorf("unexpected result, got %v, want %v", rows, exp)
		}
		if !isDone(d) {
			t.Errorf("decoder unexpectedly not done")
		}
	}
}

func TestDecode_Map(t *testing.T) {
	data := `foo,bar,baz
a,b,c
`
	exp := map[string]string{
		"foo": "a",
		"bar": "b",
		"baz": "c",
	}
	got := map[string]string{}
	d := NewDecoder(strings.NewReader(data))
	if err := d.DecodeNext(&got); err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("unexpected result, got %v, want %v", got, exp)
	}
	if !isDone(d) {
		t.Errorf("decoder unexpectedly not done")
	}
}

func TestDecode_MapErrors(t *testing.T) {
	d := NewDecoder(strings.NewReader(`foo,bar
a,b
`))

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
	d := NewDecoder(strings.NewReader(`N
128.0.0.1
`))
	var s struct{ N *net.IP }
	if err := d.DecodeNext(&s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !ip.Equal(*s.N) {
		t.Errorf("unexpected result, got %v want %v", s.N, ip)
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
