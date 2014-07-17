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
	// DecodeNext populates v with the values from the next row in the
	// Decoder's Reader.
	//
	// On the first call to DecodeNext, the first row in the reader will be
	// used as the header row to map CSV fields to struct fields, and the
	// second row will be read to populate v.
	DecodeNext(v interface{}) error
}

// DecodeOpts specifies options to modify decoding behavior.
type DecoderOpts struct {
	Comma            rune // field delimiter (set to ',' by default)
	Comment          rune // comment character for start of line
	LazyQuotes       bool // allow lazy quotes
	TrimLeadingSpace bool // trim leading space
}

type decoder struct {
	r    csv.Reader
	hm   map[string]int
	opts DecoderOpts
}

// NewDecoder returns a Decoder that reads from r.
func NewDecoder(r io.Reader) Decoder {
	return NewDecoderOpts(r, DecoderOpts{})
}

func NewDecoderOpts(r io.Reader, opts DecoderOpts) Decoder {
	csvr := csv.NewReader(r)
	if opts.Comma != rune(0) {
		csvr.Comma = opts.Comma
	}
	if opts.Comment != rune(0) {
		csvr.Comment = opts.Comment
	}
	csvr.LazyQuotes = opts.LazyQuotes
	csvr.TrimLeadingSpace = opts.TrimLeadingSpace
	return &decoder{r: *csvr, opts: opts}
}

func (d *decoder) DecodeNext(v interface{}) error {
	line, err := d.read()
	if err != nil {
		return err
	}

	// v is nil, skip this line and proceed.
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("must be pointer")
	}
	rv = rv.Elem()

	switch rv.Type().Kind() {
	case reflect.Map:
		return d.decodeMap(v, line)
	case reflect.Struct:
		return d.decodeStruct(v, line)
	default:
		return errors.New("must be pointer to struct")
	}
}
func (d *decoder) decodeMap(v interface{}, line []string) error {
	rv := reflect.ValueOf(v)
	t := rv.Elem().Type()
	if t.Key().Kind() != reflect.String {
		return errors.New("map key must be string")
	}
	switch t.Elem().Kind() {
	case reflect.String:
		m := *(v.(*map[string]string))
		for hv, hidx := range d.hm {
			m[hv] = line[hidx]
		}
	// TODO: Support arbitrary map values by parsing string values
	case reflect.Interface:
		return errors.New("TODO")
	default:
		return fmt.Errorf("can't decode type %v", t.Elem().Kind())
	}
	return nil
}

func (d *decoder) decodeStruct(v interface{}, line []string) error {
	rv := reflect.ValueOf(v).Elem()
	t := rv.Type()
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
