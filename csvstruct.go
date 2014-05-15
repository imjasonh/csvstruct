package csvstruct

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
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

func (d decoder) DecodeNext(v interface{}) error {
	if len(d.hm) == 0 {
		// First run; read header row
		header, err := d.r.Read()
		if err != nil {
			return fmt.Errorf("error reading headers: %v", err)
		}
		d.hm = reverse(header)
	}

	line, err := d.r.Read()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("must be pointer")
	}

	t := reflect.TypeOf(rv.Elem())
	out := reflect.New(t).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			continue
		}
		n := f.Name
		strv := line[d.hm[n]]
		log.Printf("%s = %s\n", n, strv)
		out.FieldByName(n).SetString(strv)
	}
	v = out
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
