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
	SkipLeadingRows  int  // number of leading rows to skip
}

type decoder struct {
	r       csv.Reader
	hm      map[string]int
	opts    DecoderOpts
	skipped int
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
	for d.skipped < d.opts.SkipLeadingRows {
		// NB: Leading rows must still have the expected number of fields.
		if _, err := d.r.Read(); err != nil {
			return err
		}
		d.skipped++
	}

	// v is nil, skip this line and proceed.
	if v == nil {
		_, err := d.read()
		return err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
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
