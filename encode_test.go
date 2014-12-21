package csvstruct

import (
	"bytes"
	"net"
	"testing"
	"time"
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
a,,
,b,
`,
	}, {
		// Encoding incomplete structs still fills in missing columns.
		[]interface{}{row{"a", "", ""}, struct{ Foo, Bar string }{"", "b"}},
		`Foo,Bar,Baz
a,,
,b,
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
			Baz     string
		}{"a", "b", "c", "d"}},
		`renamed_foo,Bar,Baz
a,b,d
`,
	}, {
		// If the first row contains no encodable fields, further rows
		// will be ignored as well, resulting in an empty output.
		[]interface{}{struct {
			Ignored    string `csv:"-"`
			unexported string
		}{"you", "won't"}, struct {
			Exported string
		}{"see"}},
		"",
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
			struct{ Baz string }{"baz"},              // Will be skipped because it shares no fields.
			struct{ Bar, Baz string }{"bar", "baz"}}, // Only shares Bar, only writes Bar.
		`Foo,Bar
foo,bar
,bar
`,
	}, {
		// Encoding rows with the same fields but with different types.
		[]interface{}{
			struct{ Foo string }{"foo"},
			struct{ Foo int64 }{123},
			struct{ Foo bool }{true}},
		`Foo
foo
123
true
`,
	}} {
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		for _, r := range c.rows {
			if err := e.EncodeNext(r); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
		if got := buf.String(); got != c.exp {
			t.Errorf("unexpected result encoding %+v, got %s, want %s", c.rows, got, c.exp)
		}
	}
}

func TestEncode_Opts(t *testing.T) {
	rows := []struct{ A, B, C string }{
		{"a", "b", "c"},
		{"d", "e", "f"}}

	for _, c := range []struct {
		opts EncodeOpts
		exp  string
	}{{
		EncodeOpts{Comma: '%'},
		`A%B%C
a%b%c
d%e%f
`,
	}, {
		EncodeOpts{SkipHeader: true},
		`a,b,c
d,e,f
`,
	}, {
		EncodeOpts{UseCRLF: true},
		"A,B,C\r\na,b,c\r\nd,e,f\r\n",
	}} {
		var buf bytes.Buffer
		e := NewEncoder(&buf).Opts(c.opts)
		for _, r := range rows {
			if err := e.EncodeNext(r); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
		if got := buf.String(); got != c.exp {
			t.Errorf("unexpected results encoding %+v, got %s, want %s", rows, got, c.exp)
		}
	}
}

func TestEncode_Map(t *testing.T) {
	for _, c := range []struct {
		rows []map[string]interface{}
		exp  string
	}{{
		[]map[string]interface{}{{
			"foo": "a",
			"bar": true,
			"baz": 1.23,
		}, {
			"foo": "b",
			"bar": false,
			"baz": 4.56,
		}},
		// Keys are sorted before being written to the header
		`bar,baz,foo
true,1.23,a
false,4.56,b
`,
	}, {
		[]map[string]interface{}{{
			"foo": "a",
		}, {
			"bar": "b",
		}},
		`foo
a
`,
	}, {
		[]map[string]interface{}{{
			"foo": "",
		}, {
			"foo": true,
		}},
		`foo

true
`,
	}} {
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		for _, r := range c.rows {
			if err := e.EncodeNext(r); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
		if got := buf.String(); got != c.exp {
			t.Errorf("unexpected results encoding %+v, got %s, want %s", c.rows, got, c.exp)
		}
	}
}

// Tests that encoding a struct then encoding a compatible map works as expected.
func TestEncode_Hybrid(t *testing.T) {
	var buf bytes.Buffer
	e := NewEncoder(&buf)
	s := struct {
		Foo string `csv:"foo"`
		Bar string
	}{"a", "b"}
	if err := e.EncodeNext(s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	m := map[string]interface{}{
		"foo": "c",
		"Bar": "d",
	}
	if err := e.EncodeNext(m); err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	exp := `foo,Bar
a,b
c,d
`
	if got := buf.String(); got != exp {
		t.Errorf("unexpected results, got %s, want %s", got, exp)
	}
}

// Tests that values that implement encoding.TextMarshaler are correctly marshaled.
func TestEncode_TextMarshaler(t *testing.T) {
	var buf bytes.Buffer
	e := NewEncoder(&buf)
	s := struct {
		T time.Time
		N net.IP
	}{time.Unix(1234567890, 0), net.IPv4(128, 0, 0, 1)}
	if err := e.EncodeNext(s); err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	exp := `T,N
2009-02-13T18:31:30-05:00,128.0.0.1
`
	if got := buf.String(); got != exp {
		t.Errorf("unexpected results, got %s, want %s", got, exp)
	}
}
