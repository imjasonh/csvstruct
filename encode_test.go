package csvstruct

import (
	"bytes"
	"testing"
)

func TestEncodeNext(t *testing.T) {
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
d,e,f
`,
	}, {
		[]row{{"a", "", ""}, {"", "b", ""}},
		`Foo,Bar,Baz
a,"",""
"",b,""
`,
	}, {
		[]row{{"a", "", ""}, {"", "b", ""}},
		`Foo,Bar,Baz
a,"",""
"",b,""
`,
	}} {
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		for _, r := range c.rows {
			if err := e.EncodeNext(r); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
		got := buf.String()
		if got != c.exp {
			t.Errorf("unexpected result, got %s, want %s", got, c.exp)
		}
	}
}

func TestEncode_Unexported(t *testing.T) {
	r := struct {
		Exported, unexported string
	}{"a", "b"}
	var buf bytes.Buffer
	if err := NewEncoder(&buf).EncodeNext(r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := `Exported
a
`
	got := buf.String()
	if got != exp {
		t.Errorf("unexpected result, got %s, want %s", got, exp)
	}
}

func TestEncode_Tags(t *testing.T) {
	type row struct {
		Foo     string `csv:"renamed_foo"`
		Bar     string
		Ignored string `csv:"-"`
	}
	var buf bytes.Buffer
	if err := NewEncoder(&buf).EncodeNext(row{"a", "b", "c"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := `renamed_foo,Bar
a,b
`
	got := buf.String()
	if got != exp {
		t.Errorf("unexpected result, got %s, want %s", got, exp)
	}
}

func TestEncode_NonStrings(t *testing.T) {
	r := struct {
		Int     int
		Int64   int64
		Uint64  uint64
		Float64 float64
		Bool    bool
	}{123, -123456789, 123456789, 123.456, true}
	var buf bytes.Buffer
	if err := NewEncoder(&buf).EncodeNext(r); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	exp := `Int,Int64,Uint64,Float64,Bool
123,-123456789,123456789,123.456000,true
`
	got := buf.String()
	if got != exp {
		t.Errorf("unexpected result, got %s, want %s", got, exp)
	}
}

func TestEncodeDifferent(t *testing.T) {
	type row1 struct {
		Foo, Bar string
	}
	type row2 struct {
		Baz string
	}
	type row3 struct {
		Bar, Baz string
	}
	var buf bytes.Buffer
	e := NewEncoder(&buf)
	// Headers are taken from the fields in the first call to EncodeNext.
	// Further calls add whatever fields they can, and if no fields are
	// shared then the row is not written.
	for _, r := range []interface{}{row1{"foo", "bar"}, row2{"baz"}, row3{"bar", "baz"}} {
		if err := e.EncodeNext(r); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
	exp := `Foo,Bar
foo,bar
"",bar
`
	got := buf.String()
	if got != exp {
		t.Errorf("unexpected result, got %s, want %s", got, exp)
	}
}
