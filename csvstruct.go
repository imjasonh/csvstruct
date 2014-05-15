// TODO: support DecodeNext(nil) to skip a line
// TODO: NewEncoder/EncodeNext -- header will be fields in first item...
// TODO: Encode/Decode map[string]string

// Package csvstruct provides methods to decode a CSV file into a struct.
package csvstruct

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// Decoder reads and decodes CSV rows from an input stream.
type Decoder interface {
	DecodeNext(v interface{}) error
}

type decoder struct {
	hm map[string]int
	r  csv.Reader
}

// NewDecoder returns a decoder that reads from r.
func NewDecoder(r io.Reader) Decoder {
	return &decoder{
		r: *csv.NewReader(r),
	}
}

// DecodeNext reads the next CSV-encoded value from its input and stores it in the value pointed to by v.
func (d *decoder) DecodeNext(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("must be pointer")
	}
	rv = rv.Elem()

	t := reflect.ValueOf(v).Elem().Type()
	if t.Kind() != reflect.Struct {
		return errors.New("must be pointer to struct")
	}

	line, err := d.read()
	if err != nil {
		return err
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			continue
		}
		n := f.Name
		if f.Tag.Get("csv") != "" {
			n = f.Tag.Get("csv")
			if n == "-" {
				continue
			}
		}
		idx, ok := d.hm[n]
		if !ok {
			// Unmapped header value
			continue
		}
		strv := line[idx]
		vf := rv.FieldByName(f.Name)
		if vf.CanSet() {
			switch vf.Kind() {
			case reflect.String:
				vf.SetString(strv)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				i, err := strconv.ParseInt(strv, 10, 64)
				if err != nil {
					return fmt.Errorf("error decoding: %v", err)
				}
				vf.SetInt(i)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				u, err := strconv.ParseUint(strv, 10, 64)
				if err != nil {
					return fmt.Errorf("error decoding: %v", err)
				}
				vf.SetUint(u)
			case reflect.Float64:
				f, err := strconv.ParseFloat(strv, 64)
				if err != nil {
					return fmt.Errorf("error decoding: %v", err)
				}
				vf.SetFloat(f)
			case reflect.Bool:
				b, err := strconv.ParseBool(strv)
				if err != nil {
					return fmt.Errorf("error decoding: %v", err)
				}
				vf.SetBool(b)
			default:
				return fmt.Errorf("can't decode type %v", vf.Type())
			}
		}
	}
	return nil
}

func (d *decoder) read() ([]string, error) {
	if d.hm == nil {
		// First run; read header row
		header, err := d.r.Read()
		if err != nil {
			return nil, fmt.Errorf("error reading headers: %v", err)
		}
		d.hm = reverse(header)
	}
	// Read data row into []string
	return d.r.Read()
}

func reverse(in []string) map[string]int {
	m := make(map[string]int, len(in))
	for i, v := range in {
		m[v] = i
	}
	return m
}
