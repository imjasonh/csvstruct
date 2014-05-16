package csvstruct

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	type row struct {
		String     string
		Renamed    string `csv:"renamerized"`
		Int64      int64
		unexported string
	}

	in := []row{{"a", "b", 123, "ignored"}, {"c", "d", 456, "ignored"}}

	var buf bytes.Buffer
	e := NewEncoder(&buf)
	for _, i := range in {
		if err := e.EncodeNext(i); err != nil {
			t.Errorf("unexpected error encoding %v: %v", i, err)
		}
	}
	exp := `String,renamerized,Int64
a,b,123
c,d,456
`
	got := buf.String()
	if got != exp {
		t.Errorf("unexpected result, got %s, want %s", got, exp)
	}

	out := []row{}
	d := NewDecoder(&buf)
	for {
		var r row
		if err := d.DecodeNext(&r); err == io.EOF {
			break
		} else if err != nil {
			t.Errorf("unexpected error decoding: %v")
		}
		out = append(out, r)
	}
	if !isDone(d) {
		t.Errorf("decoder unexpectedly not done")
	}
	// Unexported fields will not survive the roundtrip
	expRows := []row{{"a", "b", 123, ""}, {"c", "d", 456, ""}}
	if !reflect.DeepEqual(expRows, out) {
		t.Errorf("got unexpected result, got %v, want %v", out, expRows)
	}

}
