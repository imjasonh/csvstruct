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
	w  csv.Writer
	hm map[string]int
}

// NewEncoder returns an encoder that writes to w.
func NewEncoder(w io.Writer) Encoder {
	return &encoder{w: *csv.NewWriter(w)}
}

// EncodeNext writes the CSV encoding of v to the stream.
func (e *encoder) EncodeNext(v interface{}) error {
	if v == nil {
		return nil
	}

	t := reflect.ValueOf(v).Type()
	if e.hm == nil {
		e.hm = make(map[string]int)
		headers := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // Filter unexported fields
				continue
			}
			n := f.Name
			if f.Tag.Get("csv") != "" {
				n = f.Tag.Get("csv")
				if n == "-" {
					continue
				}
			}
			headers = append(headers, n)
			e.hm[n] = i
		}
		if err := e.w.Write(headers); err != nil {
			return err
		}
	}

	rv := reflect.ValueOf(v)
	row := []string{}
	add := false // Whether there has been a row to write in this call.
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // Filter unexported fields
			continue
		}
		n := f.Name
		if f.Tag.Get("csv") != "" {
			n = f.Tag.Get("csv")
		}

		fi, ok := e.hm[n]
		if !ok {
			// Unmapped header value
			continue
		}

		// Increase the row size to fit the new row.
		for fi >= len(row) {
			row = append(row, "")
		}

		add = true
		vf := rv.Field(i)
		switch vf.Kind() {
		case reflect.String:
			row[fi] = vf.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			row[fi] = fmt.Sprintf("%d", vf.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			row[fi] = fmt.Sprintf("%d", vf.Uint())
		case reflect.Float64:
			row[fi] = fmt.Sprintf("%f", vf.Float())
		case reflect.Bool:
			row[fi] = fmt.Sprintf("%t", vf.Bool())
		default:
			return fmt.Errorf("can't decode type %v", f.Type)
		}
	}
	if !add {
		return nil
	}
	if err := e.w.Write(row); err != nil {
		return err
	}
	e.w.Flush()
	return e.w.Error()
}
