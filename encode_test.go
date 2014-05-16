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
		rows []interface{}
		exp  string
	}{{
		[]interface{}{row{"a", "b", "c"}, row{"d", "e", "f"}},
		`Foo,Bar,Baz
a,b,c
d,e,f
`,
	}, {
		[]interface{}{row{"a", "", ""}, row{"", "b", ""}},
		`Foo,Bar,Baz
a,"",""
"",b,""
`,
	}, {
		[]interface{}{row{"a", "", ""}, row{"", "b", ""}},
		`Foo,Bar,Baz
a,"",""
"",b,""
`,
	}, {
		// Encoding unexported fields.
		[]interface{}{struct{ Exported, unexported string }{"a", "b"}},
		`Exported
a
`,
	}, {
		// Encoding renamed and ignored fields.
		[]interface{}{struct {
			Foo     string `csv:"renamed_foo"`
			Bar     string
			Ignored string `csv:"-"`
		}{"a", "b", "c"}},
		`renamed_foo,Bar
a,b
`,
	}, {
		// Encoding non-string fields.
		[]interface{}{struct {
			Int     int
			Int64   int64
			Uint64  uint64
			Float64 float64
			Bool    bool
		}{123, -123456789, 123456789, 123.456, true}},
		`Int,Int64,Uint64,Float64,Bool
123,-123456789,123456789,123.456000,true
`,
	}, {
		// Encoding rows with different fields.
		// Headers are taken from the fields in the first call to EncodeNext.
		// Further calls add whatever fields they can, and if no fields are
		// shared then the row is not written.
		[]interface{}{
			struct{ Foo, Bar string }{"foo", "bar"},
			struct{ Baz string }{"baz"},
			struct{ Bar, Baz string }{"bar", "baz"}},
		`Foo,Bar
foo,bar
"",bar
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
			t.Errorf("unexpected result encoding %+v, got %s, want %s", c.rows, got, c.exp)
		}
	}
}
