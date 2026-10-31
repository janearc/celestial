package stars

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/janearc/celestial/measure"
)

// mainSequenceText is pecaut and mamajek's table of mean dwarf stars, as
// mainsequence.txt says, kept as it was published.
//
//go:embed mainsequence.txt
var mainSequenceText string

// columns are the table's columns, in their order.
var columns = []string{"type", "mass", "teff", "logL", "radius"}

// column is where the named column falls in a row. Every name asked for
// is one of columns.
func column(name string) int { return slices.Index(columns, name) }

// tableLogBase is the base of the table's logarithm of luminosity.
// Defined by the table, which gives log10 of the sun's.
const tableLogBase = 10

// dwarf is a mean star of the main sequence: one of the table's rows, or
// a place between two of them.
type dwarf struct {
	kind   string
	mass   measure.Mass
	heat   measure.Temperature
	light  measure.Power
	radius measure.Length
}

// readTable is the table, lightest first.
func readTable() ([]dwarf, error) { return readDwarfs(mainSequenceText) }

// mainSequence is the table, read once and sorted lightest first, since
// the published masses rise a little in two places.
var mainSequence = sync.OnceValues(readTable)

// lighter orders two rows by mass, lightest first.
func lighter(one, other dwarf) int {
	ratio := one.mass.Over(other.mass)
	switch {
	case ratio < whole:
		return before
	case ratio > whole:
		return after
	}
	return alike
}

// How lighter orders two rows, as slices' sorting wants.
const (
	before = -1 // defined by slices.SortStableFunc: the first goes first
	alike  = 0  // defined by slices.SortStableFunc: either may go first
	after  = 1  // defined by slices.SortStableFunc: the second goes first
)

// readDwarfs is the rows of the table's text, lightest first.
func readDwarfs(text string) ([]dwarf, error) {
	var rows []dwarf
	for _, line := range strings.Split(text, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		row, err := readDwarf(strings.Fields(line))
		if err != nil {
			return nil, err
		}
		rows = append(rows, *row)
	}
	slices.SortStableFunc(rows, lighter)
	return rows, nil
}

// readDwarf is one row, from its fields in the order of columns, each in
// the table's own units: the sun's, kelvin, and log10 of the sun's. It
// refuses a row that is not one, and then gives none.
func readDwarf(fields []string) (*dwarf, error) {
	if len(fields) != len(columns) {
		return nil, refuse("a main sequence row %q", fields)
	}
	kind := fields[column("type")]
	numbers := make(map[string]float64, len(fields))
	for place, field := range fields {
		if place == column("type") {
			continue
		}
		number, err := strconv.ParseFloat(field, bits)
		if err != nil {
			name := columns[place]
			return nil, fmt.Errorf("stars: %s of %s: %w", name,
				kind, err)
		}
		numbers[columns[place]] = number
	}
	light := math.Pow(tableLogBase, numbers["logL"])
	return &dwarf{kind: kind, mass: measure.SolarMasses(numbers["mass"]),
		heat:   measure.Kelvin(numbers["teff"]),
		light:  measure.SolarLuminosities(light),
		radius: measure.SolarRadii(numbers["radius"])}, nil
}

// bits is the size of the floats the table is read into. Defined by
// float64.
const bits = 64

// ends is the lightest and the heaviest rows of the table.
func ends(table []dwarf) (lightest, heaviest dwarf) {
	return slices.MinFunc(table, lighter), slices.MaxFunc(table, lighter)
}

// along is the main sequence at a mass within the table: brightness and
// radius each found between the two rows either side, in the
// logarithms, where the main sequence runs nearly straight.
func along(table []dwarf, mass measure.Mass) dwarf {
	above, _ := slices.BinarySearchFunc(table, mass, reaches)
	upper := table[above]
	if upper.mass.Over(mass) == whole || above == firstRow {
		return upper
	}
	lower := table[above-oneRow]
	share := math.Log(mass.Over(lower.mass)) /
		math.Log(upper.mass.Over(lower.mass))
	brighter := math.Pow(upper.light.Over(lower.light), share)
	wider := math.Pow(upper.radius.Over(lower.radius), share)
	return dwarf{mass: mass, light: lower.light.Times(brighter),
		radius: lower.radius.Times(wider)}
}

// The places along steps between.
const (
	firstRow = 0 // defined by a slice's order: the first row
	oneRow   = 1 // defined by a slice's order: one row back
)

// reaches orders a row against a mass, as a binary search wants: before
// the mass, at it, or past it.
func reaches(row dwarf, mass measure.Mass) int {
	return lighter(row, dwarf{mass: mass})
}
