package gw

import (
	"fmt"
	"strings"
)

// Estimate stores a catalogue value in SI with its reported 90% credible
// interval. Bounds are offsets below and above the central value; a missing
// bound leaves the interval unknown.
type Estimate struct {
	unit         string
	value        float64
	below, above float64
	bounded      bool
}

// Value returns the estimate in the SI unit named by Unit.
func (estimate *Estimate) Value() float64 { return estimate.value }

// Unit returns kg, m, s^-1, J, W, sr, or an empty string for a dimensionless
// value.
func (estimate *Estimate) Unit() string { return estimate.unit }

// Interval returns absolute lower and upper limits and whether both bounds were
// reported. If either is missing, both returned endpoints equal Value and known
// is false.
func (estimate *Estimate) Interval() (lowest, highest float64, known bool) {
	if !estimate.bounded {
		return estimate.value, estimate.value, false
	}
	return estimate.value - estimate.below, estimate.value + estimate.above,
		true
}

// String formats the central value, lower and upper offsets, and unit.
func (estimate *Estimate) String() string {
	written := fmt.Sprintf("%g %s", estimate.value, estimate.unit)
	if estimate.bounded {
		written = fmt.Sprintf("%g -%g +%g %s", estimate.value,
			estimate.below, estimate.above, estimate.unit)
	}
	return strings.TrimSpace(written)
}
