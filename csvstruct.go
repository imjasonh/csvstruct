package csvstruct

import (
	"encoding/csv"
	"io"
	"reflect"
)

type Decoder interface {
	DecodeNext(v interface{}) error
}

type decoder struct {
	hm map[string]int
	r  csv.Reader
}

func NewDecoder(r io.Reader) Decoder {
	return decoder{
		r: *csv.NewReader(r),
	}
}

func (d decoder) DecodeNext(in interface{}) error {
	if len(d.hm) == 0 {
		// First run; read header row
		header, err := d.r.Read()
		if err != nil {
			return err
		}
		d.hm = reverse(header)
	}

	t := reflect.TypeOf(in)

	line, err := d.r.Read()
	if err != nil {
		return err
	}

	out := reflect.ValueOf(in)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		n := f.Name
		// TODO: f.Tag.Get("csv")
		val := line[d.hm[n]]
		v := reflect.ValueOf(&val)
		v.Elem().SetString(line[d.hm[n]])
		out.Elem().Set(v)
	}
	return nil
}

func reverse(in []string) map[string]int {
	m := make(map[string]int, len(in))
	for i, v := range in {
		m[v] = i
	}
	return m
}

// TODO: NewEncoder/EncodeNext -- header will be fields in first item...
// TODO: Encode/Decode map[string]interface{} ?
// TODO: Encode/Decode non-string values?
