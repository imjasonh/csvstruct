[![GoDoc](https://godoc.org/github.com/ImJasonH/csvstruct?status.png)](https://godoc.org/github.com/ImJasonH/csvstruct)
[![Build Status](https://travis-ci.org/ImJasonH/csvstruct.svg?branch=master)](https://travis-ci.org/ImJasonH/csvstruct)

This library provides methods to read and write CSV data into and from Go structs.

Decoding
-----

Given the following CSV data:
```
Name,Age,Height
Alice,25,5.7
Bob,24,5.9
```

You could decode the data into structs like so:
```
f, _ := os.Open("path/to/your.csv")
defer f.Close()
type Person struct {
	Name string
	Age int
	Height float64
}
var p Person
var err error
d := csvstruct.NewDecoder(f)
for err != io.EOF {
  err = d.DecodeNext(&p)
  if err != nil && err != io.EOF {
  	// handle error
  }
  fmt.Printf('%s's age is %d\n", p.Name, p.Age)
}
```

Encoding
-----
Similarly, given structs, you can generate CSV data.

```
people := []Person{{"Carl", 31, 6.0}, {"Debbie", 27, 5.3}}
e := csvstruct.NewEncoder(os.Stdout)
for _, p := range people {
	if err := e.EncodeNext(p); err != nil {
		// handle error
	}
}
```

Struct tags are supported to override the struct's field names and ignore fields. See the GoDoc for more information and tests for more examples.
