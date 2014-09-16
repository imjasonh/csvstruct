package csvstruct

import (
	"encoding/csv"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func BenchmarkDecode(b *testing.B) {
	in := generateCSV(numRows)
	b.ResetTimer()
	d := NewDecoder(in)
	for i := 0; i < b.N; i++ {
		var r struct{ A, B, C string }
		for {
			if err := d.DecodeNext(&r); err == io.EOF {
				break
			} else if err != nil {
				b.Errorf("unexpected error: %v", err)
				return
			}
		}
	}
}

func BenchmarkCSVRead(b *testing.B) {
	in := generateCSV(numRows)
	b.ResetTimer()
	r := csv.NewReader(in)
	for i := 0; i < b.N; i++ {
		if _, err := r.ReadAll(); err != nil {
			b.Errorf("unexpected error: %v", err)
			return
		}
	}
}

const (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	strLen   = 5
	numRows  = 10
)

var r = rand.New(rand.NewSource(time.Now().Unix()))

func randString() string {
	s := ""
	for i := 0; i < strLen; i++ {
		s += string(alphabet[r.Intn(len(alphabet))])
	}
	return s
}

// TODO: Generate the CSV on-demand instead of cramming it all into memory
func generateCSV(rows int) io.Reader {
	rs := []string{"A,B,C"}
	for i := 0; i < rows; i++ {
		rs = append(rs, strings.Join([]string{randString(), randString(), randString()}, ","))
	}
	return strings.NewReader(strings.Join(rs, "\n"))
}
