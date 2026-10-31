// Package research is every measured or picked number celestial uses,
// as research-derived-data.yml holds it: a measured one read from its
// published source, never typed from memory, and a picked one saying who
// picked it. Code asks for a number by its name and holds no copy.
//
// The sources the data names are in sources.yaml, beside it.
//
// A number true by definition is not here: it is a constant in the
// package it belongs to, and its comment says what defines it.
package research

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"
)

// fileName is the data's file, as a refusal names it.
const fileName = "research-derived-data.yml"

// raw is the data as the file holds it.
//
//go:embed research-derived-data.yml
var raw []byte

// parse reads the data.
func parse() (numbers.Numbers, error) {
	return numbers.Read(fileName, raw)
}

// parsed is the data, read once, the first time it is asked for.
var parsed = sync.OnceValues(parse)

// Read is the research data, every number by its name, or the refusal
// puzzlemouth gives for a file it cannot read. A caller reads it to say
// where a number came from.
func Read() (numbers.Numbers, error) {
	return parsed()
}

// Value is a measured number by its name. The data is part of celestial
// and a name it does not hold is a mistake in celestial's own code,
// which its tests find, so asking for one stops the program and says
// which name it was.
func Value(name string) float64 {
	read, err := Read()
	if err != nil {
		panic(fmt.Errorf("research: %w", err))
	}
	number, err := read.Get(name)
	if err != nil {
		panic(fmt.Errorf("research: %w", err))
	}
	return number.Value
}
