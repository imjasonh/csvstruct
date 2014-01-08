package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strings"
)

type example struct {
	Foo string `csv:"foo"`
	Bar string
}

func main() {
	data := `
foo,Bar,baz
a,b,c
d,e,f
g,h,i`
	ch := make(chan interface{})
	r, _ := newCsvReader(strings.NewReader(data))
	e := example{}
	_ = r.Read(e, ch)
}

type csvReader struct {
	header []string
	r      csv.Reader
}

func newCsvReader(r io.Reader) (*csvReader, error) {
	csvr := csv.NewReader(r)
	header, err := csvr.Read()
	if err != nil {
		return nil, err
	}
	c := csvReader{
		header: header,
		r:      *csvr,
	}
	return &c, nil
}

func (r csvReader) Read(in interface{}, ch chan<- interface{}) error {
	t := reflect.TypeOf(in)
	hm := reverse(r.header)

	line, err := r.r.Read()
	for err != io.EOF {
		if err != nil {
			return err
		}
		out := reflect.ValueOf(&in)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			n := f.Name
			if f.Tag.Get("csv") == "" {
				n = f.Tag.Get("csv")
			}
			val := line[hm[n]]
			v := reflect.ValueOf(&val)
			v.Elem().SetString(line[hm[n]])
			out.Elem().Set(v)
		}
		fmt.Println("%v", out.Elem().Interface())
		line, err = r.r.Read()
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