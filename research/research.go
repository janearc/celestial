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

	"github.com/janearc/puzzlemouth/numbers"
)

// data is research-derived-data.yml, built into celestial.
//
//go:embed research-derived-data.yml
var data []byte

// loaded is the data, parsed once when the program starts, and loadErr
// is why it did not parse, if it did not.
var loaded, loadErr = numbers.Read("research-derived-data.yml", data)

// Read returns every number in the research data, for a caller that
// wants to say where a number came from. returns nil and error if the
// data did not parse.
func Read() (numbers.Numbers, error) {
	if loadErr != nil {
		return nil, loadErr
	}
	return loaded, nil
}

// Value returns a number by its name, measured or picked. the data is
// built into celestial, so a name it does not hold is a bug in
// celestial's own code, which its tests find: asking for one panics and
// names it.
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
