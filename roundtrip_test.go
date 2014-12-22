package csvstruct

import (
	"bytes"
	"io"
	"net"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	type row struct {
		String     string
		Renamed    string `csv:"renamerized"`
		Int64      int64
		unexported string
		StringPtr  *string
		IP         *net.IP
	}
	s := "foo"
	ip := net.IPv4(128, 0, 0, 1)

	in := []row{{"a", "b", 123, "ignored", &s, &ip}, {"c", "d", 456, "ignored", &s, &ip}}

	var buf bytes.Buffer
	e := NewEncoder(&buf)
	for _, i := range in {
		if err := e.EncodeNext(i); err != nil {
			t.Errorf("unexpected error encoding %v: %v", i, err)
		}
	}
	want := `String,renamerized,Int64,StringPtr,IP
a,b,123,foo,128.0.0.1
c,d,456,foo,128.0.0.1
`
	got := buf.String()
	if got != want {
		t.Errorf("unexpected result, got %s, want %s", got, want)
	}

	out := []row{}
	d := NewDecoder(&buf)
	for {
		var r row
		if err := d.DecodeNext(&r); err == io.EOF {
			break
		} else if err != nil {
			t.Errorf("unexpected error decoding: %v", err)
		}
		out = append(out, r)
	}
	if !isDone(d) {
		t.Errorf("decoder unexpectedly not done")
	}
	// Unexported fields will not survive the roundtrip
	wantRows := []row{{"a", "b", 123, "", &s, &ip}, {"c", "d", 456, "", &s, &ip}}
	if !reflect.DeepEqual(wantRows, out) {
		t.Errorf("got unexpected result, got %v, want %v", out, wantRows)
	}

}
