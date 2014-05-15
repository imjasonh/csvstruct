package csvstruct

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
)

// Encoder encodes and writes CSV rows to an output stream.
type Encoder interface {
	EncodeNext(v interface{}) error
}

type encoder struct {
	w              *csv.Writer
	headersWritten bool
}

// NewEncoder returns an encoder that writes to w.
func NewEncoder(w io.Writer) Encoder {
	return &encoder{w: csv.NewWriter(w)}
}

// EncodeNext writes the CSV encoding of v to the stream.
func (e *encoder) EncodeNext(v interface{}) error {
	t := reflect.ValueOf(v).Type()
	if !e.headersWritten {
		headers := make([]string, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // Filter unexported fields
				continue
			}
			h := f.Name
			if f.Tag.Get("csv") != "" {
				h = f.Tag.Get("csv")
			}
			headers[i] = h
		}
		if err := e.w.Write(headers); err != nil {
			return err
		}
		e.headersWritten = true
	}

	rv := reflect.ValueOf(v)
	row := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // Filter unexported fields
			continue
		}
		vf := rv.Field(i)
		switch vf.Kind() {
		case reflect.String:
			row[i] = vf.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			row[i] = fmt.Sprintf("%d", vf.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			row[i] = fmt.Sprintf("%d", vf.Uint())
		case reflect.Float64:
			row[i] = fmt.Sprintf("%f", vf.Float())
		case reflect.Bool:
			row[i] = fmt.Sprintf("%t", vf.Bool())
		default:
			return fmt.Errorf("can't decode type %v", f.Type)
		}
	}
	return e.w.Write(row)
}
