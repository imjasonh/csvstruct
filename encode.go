package csvstruct

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
)

// Encoder encodes and writes CSV rows to an output stream.
type Encoder interface {
	// EncodeNext encodes v into a CSV row and writes it to the Encoder's
	// Writer.
	//
	// On the first call to EncodeNext, v's fields will be used to write the
	// header row, then v's values will be written as the second row.
	EncodeNext(v interface{}) error

	// Opts specifies options to modify encoding behavior.
	//
	// It returns the Encoder, to support chaining.
	Opts(EncodeOpts) Encoder
}

// EncodeOpts specifies options to modify encoding behavior.
type EncodeOpts struct {
	SkipHeader bool // True to skip writing the header row
	Comma      rune // Field delimiter (set to ',' by default)
	UseCRLF    bool // True to use \r\n as the line terminator
}

type encoder struct {
	w    csv.Writer
	hm   map[string]int
	opts EncodeOpts
}

// NewEncoder returns an encoder that writes to w.
func NewEncoder(w io.Writer) Encoder {
	csvw := csv.NewWriter(w)
	return &encoder{w: *csvw}
}

func (e *encoder) Opts(opts EncodeOpts) Encoder {
	if opts.Comma != rune(0) {
		e.w.Comma = opts.Comma
	}
	e.w.UseCRLF = opts.UseCRLF
	e.opts = opts
	return e
}

func (e *encoder) EncodeNext(v interface{}) error {
	if v == nil {
		return nil
	}
	switch reflect.ValueOf(v).Type().Kind() {
	case reflect.Map:
		return e.encodeMap(v)
	case reflect.Struct:
		return e.encodeStruct(v)
	default:
		return errors.New("must encode map or struct")
	}
}

func (e *encoder) encodeMap(v interface{}) error {
	if reflect.ValueOf(v).Type().Key().Kind() != reflect.String {
		return errors.New("map key must be string")
	}
	m := v.(map[string]interface{})

	if e.hm == nil {
		e.hm = make(map[string]int)
		headers := []string{}
		for k, _ := range m {
			headers = append(headers, k)
		}
		sort.Strings(headers)
		for i, h := range headers {
			e.hm[h] = i
		}
		if len(e.hm) == 0 {
			// First row was an empty map, so write nothing.
			// This will result in an empty output no matter what is Encoded.
			return nil
		}
		if !e.opts.SkipHeader {
			if err := e.w.Write(headers); err != nil {
				return err
			}
		}
	}
	row := make([]string, len(m))
	add := false // Whether there has been a row to write in this call.
	for h, i := range e.hm {
		val, ok := m[h]
		if !ok {
			continue
		}
		add = true
		row[i] = fmt.Sprint(val)
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

func (e *encoder) encodeStruct(v interface{}) error {
	t := reflect.ValueOf(v).Type()
	if e.hm == nil {
		e.hm = make(map[string]int)
		headers := []string{}
		i := 0
		for j := 0; j < t.NumField(); j++ {
			f := t.Field(j)
			if f.Anonymous {
				continue
			}
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
			i++
		}
		if len(e.hm) == 0 {
			// Header row has no exported, unignored fields, so write nothing.
			// This will result in an empty output no matter what is Encoded.
			return nil
		}
		if !e.opts.SkipHeader {
			if err := e.w.Write(headers); err != nil {
				return err
			}
		}
	}

	rv := reflect.ValueOf(v)
	row := make([]string, len(e.hm))
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
