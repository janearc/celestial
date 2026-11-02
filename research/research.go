// Package research supplies named measured and chosen values from
// research-derived-data.yml. Measured values identify published sources; chosen
// parameters identify their authors. sources.yaml records the works those
// entries cite.
//
// Callers request values by name rather than copying them into code. Exact
// definitions instead remain constants beside the formulas or units they
// define, with comments explaining their origin.
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

// Read returns the recorded values and provenance, or the error from parsing
// the embedded data.
func Read() (numbers.Numbers, error) {
	if loadErr != nil {
		return nil, loadErr
	}
	return loaded, nil
}

// Value looks up a measured or chosen parameter by name. Missing names and
// invalid embedded data are internal errors: Value panics with the cause so the
// project’s tests can identify them.
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
